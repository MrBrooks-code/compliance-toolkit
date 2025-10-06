package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	serviceName        = "ComplianceToolkitClient"
	serviceDisplayName = "Compliance Toolkit Client"
	serviceDescription = "Automated compliance scanning and reporting client for Windows registry security checks"
)

// complianceService implements the svc.Handler interface
type complianceService struct {
	config *ClientConfig
	logger *slog.Logger
	elog   *eventlog.Log
}

// Execute is called by the service control manager when the service starts
func (s *complianceService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	// Tell SCM we're starting
	changes <- svc.Status{State: svc.StartPending}

	// Log service start
	s.elog.Info(1, "Compliance Toolkit Client service starting")
	s.logger.Info("Service starting",
		"client_id", s.config.Client.ID,
		"hostname", s.config.Client.Hostname,
	)

	// Start the compliance client in a goroutine
	clientDone := make(chan error, 1)
	client := NewComplianceClient(s.config, s.logger)

	go func() {
		clientDone <- client.Run()
	}()

	// Tell SCM we're running
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	s.elog.Info(1, "Compliance Toolkit Client service started successfully")

	// Service control loop
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus

			case svc.Stop, svc.Shutdown:
				s.elog.Info(1, "Service stop requested")
				s.logger.Info("Service stopping")

				// Tell SCM we're stopping
				changes <- svc.Status{State: svc.StopPending}

				// Stop the client (if it has a Stop method, we'll add that)
				// For now, we'll rely on context cancellation in the scheduler

				// Wait for client to finish (with timeout)
				select {
				case err := <-clientDone:
					if err != nil {
						s.elog.Error(1, fmt.Sprintf("Client error: %v", err))
						s.logger.Error("Client error", "error", err)
						return false, 1
					}
				case <-time.After(30 * time.Second):
					s.elog.Warning(1, "Client shutdown timeout")
					s.logger.Warn("Client shutdown timeout")
				}

				s.elog.Info(1, "Service stopped")
				s.logger.Info("Service stopped")
				return false, 0

			default:
				s.elog.Warning(1, fmt.Sprintf("Unexpected control request: %v", c))
			}

		case err := <-clientDone:
			// Client finished unexpectedly
			if err != nil {
				s.elog.Error(1, fmt.Sprintf("Client finished with error: %v", err))
				s.logger.Error("Client finished with error", "error", err)
				return false, 1
			}
			s.elog.Info(1, "Client finished")
			s.logger.Info("Client finished")
			return false, 0
		}
	}
}

// runService runs the application as a Windows service
func runService(config *ClientConfig, logger *slog.Logger) error {
	// Open event log
	elog, err := eventlog.Open(serviceName)
	if err != nil {
		return fmt.Errorf("failed to open event log: %w", err)
	}
	defer elog.Close()

	elog.Info(1, "Starting service")

	// Create service handler
	service := &complianceService{
		config: config,
		logger: logger,
		elog:   elog,
	}

	// Run the service
	err = svc.Run(serviceName, service)
	if err != nil {
		elog.Error(1, fmt.Sprintf("Service failed: %v", err))
		return fmt.Errorf("service execution failed: %w", err)
	}

	return nil
}

// isWindowsService checks if the process is running as a Windows service
func isWindowsService() (bool, error) {
	return svc.IsWindowsService()
}

// installService installs the compliance client as a Windows service
func installService(configPath string) error {
	// Get executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Open service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Check if service already exists
	s, err := m.OpenService(serviceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", serviceName)
	}

	// Build service arguments
	args := []string{}
	if configPath != "" {
		absPath, err := filepath.Abs(configPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute config path: %w", err)
		}
		args = append(args, "--config", absPath)
	}

	// Create service
	s, err = m.CreateService(serviceName, exePath, mgr.Config{
		DisplayName:  serviceDisplayName,
		Description:  serviceDescription,
		StartType:    mgr.StartAutomatic,
		Dependencies: []string{"Tcpip"}, // Require network
	}, args...)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer s.Close()

	// Set service recovery options (restart on failure)
	err = s.SetRecoveryActions([]mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 120 * time.Second},
	}, 86400) // Reset failure count after 24 hours
	if err != nil {
		// Non-fatal - service is still installed
		fmt.Printf("Warning: Could not set recovery options: %v\n", err)
	}

	// Setup event log
	err = eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		// Remove service if event log setup fails
		s.Delete()
		return fmt.Errorf("failed to setup event log: %w", err)
	}

	fmt.Printf("Service %s installed successfully\n", serviceName)
	fmt.Printf("Start with: sc start %s\n", serviceName)
	return nil
}

// uninstallService removes the compliance client service
func uninstallService() error {
	// Open service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open service
	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s not found", serviceName)
	}
	defer s.Close()

	// Check if service is running
	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("failed to query service status: %w", err)
	}

	if status.State != svc.Stopped {
		// Stop the service
		fmt.Printf("Stopping service...\n")
		_, err = s.Control(svc.Stop)
		if err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}

		// Wait for service to stop (up to 30 seconds)
		timeout := time.Now().Add(30 * time.Second)
		for status.State != svc.Stopped {
			if time.Now().After(timeout) {
				return fmt.Errorf("timeout waiting for service to stop")
			}
			time.Sleep(300 * time.Millisecond)
			status, err = s.Query()
			if err != nil {
				return fmt.Errorf("failed to query service status: %w", err)
			}
		}
		fmt.Printf("Service stopped\n")
	}

	// Delete service
	err = s.Delete()
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	// Remove event log
	err = eventlog.Remove(serviceName)
	if err != nil {
		// Non-fatal
		fmt.Printf("Warning: Could not remove event log: %v\n", err)
	}

	fmt.Printf("Service %s uninstalled successfully\n", serviceName)
	return nil
}

// startService starts the compliance client service
func startService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s not found", serviceName)
	}
	defer s.Close()

	err = s.Start()
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	fmt.Printf("Service %s started successfully\n", serviceName)
	return nil
}

// stopService stops the compliance client service
func stopService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s not found", serviceName)
	}
	defer s.Close()

	status, err := s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Wait for service to stop
	timeout := time.Now().Add(30 * time.Second)
	for status.State != svc.Stopped {
		if time.Now().After(timeout) {
			return fmt.Errorf("timeout waiting for service to stop")
		}
		time.Sleep(300 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("failed to query service status: %w", err)
		}
	}

	fmt.Printf("Service %s stopped successfully\n", serviceName)
	return nil
}

// serviceStatus displays the status of the compliance client service
func serviceStatus() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s not found", serviceName)
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("failed to query service status: %w", err)
	}

	config, err := s.Config()
	if err != nil {
		return fmt.Errorf("failed to query service config: %w", err)
	}

	fmt.Printf("Service: %s\n", serviceName)
	fmt.Printf("Display Name: %s\n", config.DisplayName)
	fmt.Printf("Description: %s\n", config.Description)
	fmt.Printf("State: %s\n", getStateString(status.State))
	fmt.Printf("Start Type: %s\n", getStartTypeString(config.StartType))
	fmt.Printf("Executable: %s\n", config.BinaryPathName)

	return nil
}

// getStateString converts service state to string
func getStateString(state svc.State) string {
	switch state {
	case svc.Stopped:
		return "Stopped"
	case svc.StartPending:
		return "Start Pending"
	case svc.StopPending:
		return "Stop Pending"
	case svc.Running:
		return "Running"
	case svc.ContinuePending:
		return "Continue Pending"
	case svc.PausePending:
		return "Pause Pending"
	case svc.Paused:
		return "Paused"
	default:
		return fmt.Sprintf("Unknown (%d)", state)
	}
}

// getStartTypeString converts start type to string
func getStartTypeString(startType uint32) string {
	switch startType {
	case mgr.StartAutomatic:
		return "Automatic"
	case mgr.StartManual:
		return "Manual"
	case mgr.StartDisabled:
		return "Disabled"
	default:
		return fmt.Sprintf("Unknown (%d)", startType)
	}
}
