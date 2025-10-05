package mocks

import "compliancetoolkit/pkg"

// MockUIService is a mock implementation of UIService for testing
type MockUIService struct {
	ShowHeaderFunc             func()
	ShowMainMenuFunc           func() int
	ShowReportMenuDynamicFunc  func(reports []pkg.ReportInfo) int
	ShowErrorFunc              func(message string)
	ShowSuccessFunc            func(message string)
	ShowInfoFunc               func(message string)
	ShowProgressFunc           func(message string)
	PauseFunc                  func()
	GetIntInputFunc            func() int
	GetStringInputFunc         func() string
	ConfirmFunc                func(message string) bool
	ShowAboutFunc              func()
}

// ShowHeader mocks the ShowHeader method
func (m *MockUIService) ShowHeader() {
	if m.ShowHeaderFunc != nil {
		m.ShowHeaderFunc()
	}
}

// ShowMainMenu mocks the ShowMainMenu method
func (m *MockUIService) ShowMainMenu() int {
	if m.ShowMainMenuFunc != nil {
		return m.ShowMainMenuFunc()
	}
	return 0
}

// ShowReportMenuDynamic mocks the ShowReportMenuDynamic method
func (m *MockUIService) ShowReportMenuDynamic(reports []pkg.ReportInfo) int {
	if m.ShowReportMenuDynamicFunc != nil {
		return m.ShowReportMenuDynamicFunc(reports)
	}
	return 0
}

// ShowError mocks the ShowError method
func (m *MockUIService) ShowError(message string) {
	if m.ShowErrorFunc != nil {
		m.ShowErrorFunc(message)
	}
}

// ShowSuccess mocks the ShowSuccess method
func (m *MockUIService) ShowSuccess(message string) {
	if m.ShowSuccessFunc != nil {
		m.ShowSuccessFunc(message)
	}
}

// ShowInfo mocks the ShowInfo method
func (m *MockUIService) ShowInfo(message string) {
	if m.ShowInfoFunc != nil {
		m.ShowInfoFunc(message)
	}
}

// ShowProgress mocks the ShowProgress method
func (m *MockUIService) ShowProgress(message string) {
	if m.ShowProgressFunc != nil {
		m.ShowProgressFunc(message)
	}
}

// Pause mocks the Pause method
func (m *MockUIService) Pause() {
	if m.PauseFunc != nil {
		m.PauseFunc()
	}
}

// GetIntInput mocks the GetIntInput method
func (m *MockUIService) GetIntInput() int {
	if m.GetIntInputFunc != nil {
		return m.GetIntInputFunc()
	}
	return 0
}

// GetStringInput mocks the GetStringInput method
func (m *MockUIService) GetStringInput() string {
	if m.GetStringInputFunc != nil {
		return m.GetStringInputFunc()
	}
	return ""
}

// Confirm mocks the Confirm method
func (m *MockUIService) Confirm(message string) bool {
	if m.ConfirmFunc != nil {
		return m.ConfirmFunc(message)
	}
	return false
}

// ShowAbout mocks the ShowAbout method
func (m *MockUIService) ShowAbout() {
	if m.ShowAboutFunc != nil {
		m.ShowAboutFunc()
	}
}
