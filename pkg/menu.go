package pkg

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Menu represents an interactive CLI menu
type Menu struct {
	scanner *bufio.Scanner
}

// NewMenu creates a new menu instance
func NewMenu() *Menu {
	return &Menu{
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// ShowHeader displays the application header
func (m *Menu) ShowHeader() {
	m.Clear()
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                                      ║")
	fmt.Println("║           ╔═╗╔═╗╔╦╗╔═╗╦  ╦╔═╗╔╗╔╔═╗╔═╗  ╔╦╗╔═╗╔═╗╦  ╦╔═╦╔╦╗          ║")
	fmt.Println("║           ║  ║ ║║║║╠═╝║  ║╠═╣║║║║  ║╣    ║ ║ ║║ ║║  ╠╩╗║ ║           ║")
	fmt.Println("║           ╚═╝╚═╝╩ ╩╩  ╩═╝╩╩ ╩╝╚╝╚═╝╚═╝   ╩ ╚═╝╚═╝╩═╝╩ ╩╩ ╩           ║")
	fmt.Println("║                                                                      ║")
	fmt.Println("║                 Windows Registry Compliance Scanner                  ║")
	fmt.Println("║                          Version 1.0.0                               ║")
	fmt.Println("║                                                                      ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// ShowMainMenu displays the main menu and returns the user's choice
func (m *Menu) ShowMainMenu() int {
	m.ShowHeader()
	fmt.Println("┌───────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│                            MAIN MENU                                  │")
	fmt.Println("├───────────────────────────────────────────────────────────────────────┤")
	fmt.Println("│                                                                       │")
	fmt.Println("│       [1]  Run Reports                                                │")
	fmt.Println("│       [2]  View HTML Reports                                          │")
	fmt.Println("│       [3]  View Evidence Logs                                         │")
	fmt.Println("│       [4]  View Log Files                                             │")
	fmt.Println("│       [5]  Configuration                                              │")
	fmt.Println("│       [6]  About                                                      │")
	fmt.Println("│                                                                       │")
	fmt.Println("│       [0]  Exit                                                       │")
	fmt.Println("│                                                                       │")
	fmt.Println("└───────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Print("  ➤  Select option: ")

	return m.GetIntInput()
}

// ReportInfo represents a report configuration for menu display
type ReportInfo struct {
	Title      string
	ConfigFile string
	Category   string
	Version    string
}

// ShowReportMenuDynamic displays the report selection menu with dynamically loaded reports
func (m *Menu) ShowReportMenuDynamic(reports []ReportInfo) int {
	m.ShowHeader()
	fmt.Println("┌──────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│                        AVAILABLE REPORTS                             │")
	fmt.Println("├──────────────────────────────────────────────────────────────────────┤")
	fmt.Println("│                                                                      │")

	// Display each report dynamically
	for i, report := range reports {
		title := report.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		version := ""
		if report.Version != "" {
			version = fmt.Sprintf(" v%s", report.Version)
		}
		fmt.Printf("│      [%d]  %-50s%8s │\n", i+1, title, version)
	}

	fmt.Println("│                                                                      │")
	fmt.Printf("│      [%d]  Run ALL Reports                                            │\n", len(reports)+1)
	fmt.Println("│                                                                      │")
	fmt.Println("│      [0]  Back to Main Menu                                          │")
	fmt.Println("│                                                                      │")
	fmt.Println("└──────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Print("  ➤  Select report: ")

	return m.GetIntInput()
}

// ShowReportMenu displays the report selection menu (DEPRECATED - use ShowReportMenuDynamic)
func (m *Menu) ShowReportMenu() int {
	m.ShowHeader()
	fmt.Println("┌──────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│                        AVAILABLE REPORTS                             │")
	fmt.Println("├──────────────────────────────────────────────────────────────────────┤")
	fmt.Println("│                                                                      │")
	fmt.Println("│      [1]  System Information Report                                  │")
	fmt.Println("│      [2]  Security Audit Report                                      │")
	fmt.Println("│      [3]  Software Inventory Report                                  │")
	fmt.Println("│      [4]  Network Configuration Report                               │")
	fmt.Println("│      [5]  User Settings Report                                       │")
	fmt.Println("│      [6]  Performance Diagnostics Report                             │")
	fmt.Println("│                                                                      │")
	fmt.Println("│      [7]  Run ALL Reports                                            │")
	fmt.Println("│                                                                      │")
	fmt.Println("│      [0]  Back to Main Menu                                          │")
	fmt.Println("│                                                                      │")
	fmt.Println("└──────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Print("  ➤  Select report: ")

	return m.GetIntInput()
}

// ShowConfigMenu displays the configuration menu
func (m *Menu) ShowConfigMenu() int {
	m.ShowHeader()
	fmt.Println("CONFIGURATION")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("  [1] Set Output Directory")
	fmt.Println("  [2] Set Log Level (INFO/DEBUG)")
	fmt.Println("  [3] Set Operation Timeout")
	fmt.Println("  [4] View Current Settings")
	fmt.Println()
	fmt.Println("  [0] Back to Main Menu")
	fmt.Println()
	fmt.Print("Select option: ")

	return m.GetIntInput()
}

// ShowHTMLReportsMenu displays available HTML reports
func (m *Menu) ShowHTMLReportsMenu(reports []string) int {
	m.ShowHeader()
	fmt.Println("HTML REPORTS")
	fmt.Println("─────────────────────────────────────────────────────────────")

	if len(reports) == 0 {
		fmt.Println("  No HTML reports found.")
		fmt.Println()
		fmt.Println("  Generate reports from the 'Run Reports' menu first.")
	} else {
		for i, report := range reports {
			fmt.Printf("  [%d] %s\n", i+1, report)
		}
	}

	fmt.Println("  [0] Back to Main Menu")
	fmt.Println()
	fmt.Print("Select report to open: ")

	return m.GetIntInput()
}

// ShowLogFilesMenu displays available log files
func (m *Menu) ShowLogFilesMenu(logs []string) int {
	m.ShowHeader()
	fmt.Println("LOG FILES")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println()

	if len(logs) == 0 {
		fmt.Println("  No log files found.")
	} else {
		for i, log := range logs {
			fmt.Printf("  [%d] %s\n", i+1, log)
		}
	}

	fmt.Println()
	fmt.Println("  [0] Back to Main Menu")
	fmt.Println()
	fmt.Print("Select log to view: ")

	return m.GetIntInput()
}

// ShowAbout displays the about screen
func (m *Menu) ShowAbout() {
	m.ShowHeader()
	fmt.Println("ABOUT COMPLIANCE TOOLKIT")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("  Version:     1.0.0")
	fmt.Println("  Purpose:     Windows System Compliance Scanning")
	fmt.Println("  Platform:    Windows (x64)")
	fmt.Println()
	fmt.Println("FEATURES:")
	fmt.Println("  • Read-only registry access (defensive security)")
	fmt.Println("  • Context-aware operations with timeout protection")
	fmt.Println("  • Structured logging and HTML report generation")
	fmt.Println("  • Multiple compliance report templates")
	fmt.Println("  • Batch operations for performance")
	fmt.Println()
	fmt.Println("SECURITY:")
	fmt.Println("  • No write operations to registry")
	fmt.Println("  • Timeout protection against hanging")
	fmt.Println("  • Comprehensive error handling")
	fmt.Println()
	m.Pause()
}

// GetIntInput reads an integer from stdin
func (m *Menu) GetIntInput() int {
	if m.scanner.Scan() {
		text := strings.TrimSpace(m.scanner.Text())
		num, err := strconv.Atoi(text)
		if err != nil {
			return -1
		}
		return num
	}
	return -1
}

// GetStringInput reads a string from stdin
func (m *Menu) GetStringInput() string {
	if m.scanner.Scan() {
		return strings.TrimSpace(m.scanner.Text())
	}
	return ""
}

// Confirm asks for yes/no confirmation
func (m *Menu) Confirm(message string) bool {
	fmt.Printf("%s (y/n): ", message)
	response := m.GetStringInput()
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}

// Pause waits for user to press enter
func (m *Menu) Pause() {
	fmt.Println()
	fmt.Print("Press ENTER to continue...")
	m.scanner.Scan()
}

// ShowError displays an error message
func (m *Menu) ShowError(message string) {
	fmt.Println()
	fmt.Printf("❌ ERROR: %s\n", message)
	m.Pause()
}

// ShowSuccess displays a success message
func (m *Menu) ShowSuccess(message string) {
	fmt.Println()
	fmt.Printf("✅ SUCCESS: %s\n", message)
}

// ShowInfo displays an info message
func (m *Menu) ShowInfo(message string) {
	fmt.Println()
	fmt.Printf("ℹ INFO: %s\n", message)
}

// ShowProgress displays a progress message
func (m *Menu) ShowProgress(message string) {
	fmt.Printf("⏳ %s...\n", message)
}

// Clear clears the screen (Windows)
func (m *Menu) Clear() {
	// Simple clear - print newlines
	fmt.Print("\033[H\033[2J")
}
