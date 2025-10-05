package pkg

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// FileServiceImpl implements FileService interface
type FileServiceImpl struct{}

// NewFileService creates a new file service
func NewFileService() FileService {
	return &FileServiceImpl{}
}

// FindReportsDirectory looks for the reports directory in multiple locations
func (fs *FileServiceImpl) FindReportsDirectory(exeDir string) string {
	// Try these locations in order:
	locations := []string{
		"configs/reports",                                  // 1. Current working directory
		filepath.Join(exeDir, "configs/reports"),           // 2. Next to executable
		filepath.Join(exeDir, "..", "configs/reports"),     // 3. One level up from exe
	}

	for _, loc := range locations {
		absPath, err := filepath.Abs(loc)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			return absPath
		}
	}

	// Default fallback
	return "configs/reports"
}

// ResolveDirectory converts relative paths to absolute paths based on exe location
func (fs *FileServiceImpl) ResolveDirectory(dir, exeDir string) string {
	// If already absolute, return as-is
	if filepath.IsAbs(dir) {
		return dir
	}

	// Try current working directory first
	if _, err := os.Stat(dir); err == nil {
		absPath, _ := filepath.Abs(dir)
		return absPath
	}

	// Otherwise use path relative to executable
	return filepath.Join(exeDir, dir)
}

// ListReports lists all available reports in a directory
func (fs *FileServiceImpl) ListReports(reportsDir string) ([]ReportInfo, error) {
	files, err := os.ReadDir(reportsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read reports directory '%s': %w (try running from the directory containing the executable)", reportsDir, err)
	}

	var reports []ReportInfo
	configService := NewConfigService()

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		// Load the config to get metadata
		configPath := filepath.Join(reportsDir, file.Name())
		config, err := configService.LoadConfig(configPath)
		if err != nil {
			slog.Warn("Failed to load report config", "file", file.Name(), "error", err)
			continue
		}

		// Use metadata title, fallback to filename
		title := config.Metadata.ReportTitle
		if title == "" {
			title = file.Name()
		}

		reports = append(reports, ReportInfo{
			Title:      title,
			ConfigFile: file.Name(),
			Category:   config.Metadata.Category,
			Version:    config.Metadata.ReportVersion,
		})
	}

	return reports, nil
}

// OpenBrowser opens a URL in the default browser
func (fs *FileServiceImpl) OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Use "" as first arg to prevent issues with paths containing spaces
		cmd = exec.Command("cmd", "/c", "start", "", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}

// OpenFile opens a file with the default program
func (fs *FileServiceImpl) OpenFile(filePath string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Use explorer to open file with default program
		cmd = exec.Command("explorer", filePath)
	case "darwin":
		cmd = exec.Command("open", filePath)
	default: // linux
		cmd = exec.Command("xdg-open", filePath)
	}

	return cmd.Start()
}
