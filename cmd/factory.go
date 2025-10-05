package main

import "compliancetoolkit/pkg"

// ServiceFactory creates service instances with proper dependencies
type ServiceFactory struct {
	deps *Dependencies
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(deps *Dependencies) *ServiceFactory {
	return &ServiceFactory{deps: deps}
}

// CreateReportService creates a new report service with all dependencies
func (f *ServiceFactory) CreateReportService(title, outputDir string) pkg.ReportService {
	return pkg.NewHTMLReport(
		title,
		outputDir,
		f.deps.Logger,
		f.deps.RegistryService,
	)
}

// CreateEvidenceService creates a new evidence service with all dependencies
func (f *ServiceFactory) CreateEvidenceService(evidenceDir, reportType string) (pkg.EvidenceService, error) {
	return pkg.NewEvidenceLogger(
		evidenceDir,
		reportType,
		f.deps.Logger,
	)
}
