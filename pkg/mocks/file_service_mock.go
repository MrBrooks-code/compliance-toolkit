package mocks

import "compliancetoolkit/pkg"

// MockFileService is a mock implementation of FileService for testing
type MockFileService struct {
	FindReportsDirectoryFunc func(exeDir string) string
	ResolveDirectoryFunc     func(dir, exeDir string) string
	ListReportsFunc          func(reportsDir string) ([]pkg.ReportInfo, error)
	OpenBrowserFunc          func(url string) error
	OpenFileFunc             func(filePath string) error
}

// FindReportsDirectory mocks the FindReportsDirectory method
func (m *MockFileService) FindReportsDirectory(exeDir string) string {
	if m.FindReportsDirectoryFunc != nil {
		return m.FindReportsDirectoryFunc(exeDir)
	}
	return "configs/reports"
}

// ResolveDirectory mocks the ResolveDirectory method
func (m *MockFileService) ResolveDirectory(dir, exeDir string) string {
	if m.ResolveDirectoryFunc != nil {
		return m.ResolveDirectoryFunc(dir, exeDir)
	}
	return dir
}

// ListReports mocks the ListReports method
func (m *MockFileService) ListReports(reportsDir string) ([]pkg.ReportInfo, error) {
	if m.ListReportsFunc != nil {
		return m.ListReportsFunc(reportsDir)
	}
	return []pkg.ReportInfo{}, nil
}

// OpenBrowser mocks the OpenBrowser method
func (m *MockFileService) OpenBrowser(url string) error {
	if m.OpenBrowserFunc != nil {
		return m.OpenBrowserFunc(url)
	}
	return nil
}

// OpenFile mocks the OpenFile method
func (m *MockFileService) OpenFile(filePath string) error {
	if m.OpenFileFunc != nil {
		return m.OpenFileFunc(filePath)
	}
	return nil
}
