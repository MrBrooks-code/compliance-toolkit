# Dependency Injection Quick Reference

## TL;DR

**Goal**: Replace direct instantiation with dependency injection to enable testing, reduce coupling, and improve maintainability.

**Timeline**: 6 weeks (phased approach)

**Effort**: ~80-110 person-days

**Risk**: Low (incremental migration, full rollback plan)

---

## Key Changes at a Glance

### Before (Current)
```go
// ❌ BAD: Direct instantiation, global state
app.reader = pkg.NewRegistryReader(pkg.WithLogger(logger))
slog.SetDefault(logger) // GLOBAL STATE

htmlReport := pkg.NewHTMLReport(reportName, outputDir)
htmlReport.SetRegistryReader(app.reader) // SETTER INJECTION
```

### After (Target)
```go
// ✅ GOOD: Constructor injection, no global state
deps := NewDependencies(config, logger)
app := NewApp(deps)

reportService := factory.CreateReportService(title, outputDir)
// All dependencies passed via constructor
```

---

## Interface Definitions

### Location: `pkg/interfaces.go`

```go
// Registry operations
type RegistryService interface {
    ReadString(ctx, rootKey, path, valueName) (string, error)
    ReadValue(ctx, rootKey, path, valueName) (string, error)
    BatchRead(ctx, rootKey, path, values) (map[string]interface{}, error)
    // ... other methods
}

// Report generation
type ReportService interface {
    Generate() error
    AddResult(name, description, value, err)
    SetMetadata(metadata)
    GetOutputPath() string
}

// Evidence logging
type EvidenceService interface {
    GatherMachineInfo(RegistryService) error
    LogResult(checkName, description, regPath, valueName, actualValue, err)
    Finalize() error
    GetLogPath() string
}

// User interface
type UIService interface {
    ShowMainMenu() int
    ShowError(string)
    ShowSuccess(string)
    Pause()
    // ... other methods
}

// Configuration management
type ConfigService interface {
    LoadConfig(path) (*Config, error)
    ParseRootKey(string) (registry.Key, error)
}

// File operations
type FileService interface {
    FindReportsDirectory(exeDir) string
    ListReports(reportsDir) ([]ReportInfo, error)
    OpenBrowser(url) error
    OpenFile(path) error
}
```

---

## Dependencies Container

### Location: `cmd/dependencies.go`

```go
// All application dependencies in one place
type Dependencies struct {
    Logger          *slog.Logger
    Config          *AppConfig
    RegistryService pkg.RegistryService
    UIService       pkg.UIService
    ConfigService   pkg.ConfigService
    FileService     pkg.FileService
}

// Create and wire dependencies
func NewDependencies(config *AppConfig, logger *slog.Logger) *Dependencies {
    return &Dependencies{
        Logger:          logger,
        Config:          config,
        RegistryService: pkg.NewRegistryReader(
            pkg.WithLogger(logger),
            pkg.WithTimeout(config.Timeout),
        ),
        UIService:     pkg.NewMenu(),
        ConfigService: pkg.NewConfigService(),
        FileService:   pkg.NewFileService(),
    }
}

// Validate all dependencies are set
func (d *Dependencies) Validate() error {
    if d.Logger == nil { return errors.New("logger required") }
    if d.Config == nil { return errors.New("config required") }
    // ... validate all fields
    return nil
}
```

---

## Service Factory

### Location: `cmd/factory.go`

```go
// Creates service instances with dependencies
type ServiceFactory struct {
    deps *Dependencies
}

func NewServiceFactory(deps *Dependencies) *ServiceFactory {
    return &ServiceFactory{deps: deps}
}

// Create report service with all dependencies
func (f *ServiceFactory) CreateReportService(title, outputDir string) pkg.ReportService {
    return pkg.NewHTMLReport(
        title,
        outputDir,
        f.deps.Logger,        // ← Injected
        f.deps.RegistryService, // ← Injected
    )
}

// Create evidence service with all dependencies
func (f *ServiceFactory) CreateEvidenceService(evidenceDir, reportType string) (pkg.EvidenceService, error) {
    return pkg.NewEvidenceLogger(
        evidenceDir,
        reportType,
        f.deps.Logger, // ← Injected
    )
}
```

---

## Updated Constructors

### HTMLReport (pkg/htmlreport.go)

```go
// BEFORE ❌
func NewHTMLReport(title, outputDir string) *HTMLReport {
    return &HTMLReport{
        Title:      title,
        OutputPath: filepath.Join(outputDir, filename),
        Results:    make(map[string]ReportResult),
        // registryReader set later via SetRegistryReader() ❌
    }
}

// AFTER ✅
func NewHTMLReport(
    title, outputDir string,
    logger *slog.Logger,
    registryReader RegistryService,
) *HTMLReport {
    return &HTMLReport{
        Title:          title,
        OutputPath:     filepath.Join(outputDir, filename),
        Results:        make(map[string]ReportResult),
        logger:         logger,         // ✅ Injected
        registryReader: registryReader, // ✅ Injected
    }
}
```

### EvidenceLogger (pkg/evidence.go)

```go
// BEFORE ❌
func NewEvidenceLogger(logDir, reportType string) (*EvidenceLogger, error) {
    return &EvidenceLogger{
        LogPath:   logPath,
        StartTime: timestamp,
        Evidence:  evidence,
        // logger not stored ❌
    }, nil
}

// AFTER ✅
func NewEvidenceLogger(
    logDir, reportType string,
    logger *slog.Logger,
) (*EvidenceLogger, error) {
    return &EvidenceLogger{
        LogPath:   logPath,
        StartTime: timestamp,
        Evidence:  evidence,
        logger:    logger, // ✅ Injected
    }, nil
}
```

---

## App Refactoring

### Location: `cmd/toolkit.go`

```go
// BEFORE ❌
type App struct {
    menu        *pkg.Menu
    reader      *pkg.RegistryReader
    config      *AppConfig
    outputDir   string
    logsDir     string
    // ... many fields
}

func (app *App) init() {
    // Create reader here ❌
    app.reader = pkg.NewRegistryReader(...)
    slog.SetDefault(logger) // ❌ GLOBAL STATE
}

// AFTER ✅
type App struct {
    deps    *Dependencies    // ✅ Single dependency
    factory *ServiceFactory  // ✅ For creating services
}

func NewApp(deps *Dependencies) (*App, error) {
    if err := deps.Validate(); err != nil {
        return nil, err
    }
    return &App{
        deps:    deps,
        factory: NewServiceFactory(deps),
    }, nil
}
// ✅ NO init() method - dependencies injected
```

---

## Report Runner

### Location: `cmd/report_runner.go`

```go
// NEW: Dedicated service for running reports
type ReportRunner struct {
    deps    *Dependencies
    factory *ServiceFactory
}

func NewReportRunner(deps *Dependencies) *ReportRunner {
    return &ReportRunner{
        deps:    deps,
        factory: NewServiceFactory(deps),
    }
}

func (rr *ReportRunner) ExecuteReport(configFile string, quiet bool) error {
    // 1. Load config
    config, err := rr.deps.ConfigService.LoadConfig(configPath)

    // 2. Create services via factory
    reportSvc := rr.factory.CreateReportService(reportName, outputDir)
    evidenceSvc, _ := rr.factory.CreateEvidenceService(evidenceDir, reportType)

    // 3. Execute queries using deps.RegistryService
    for _, query := range config.Queries {
        value, err := rr.deps.RegistryService.ReadValue(ctx, rootKey, path, valueName)
        reportSvc.AddResultWithDetails(...)
        evidenceSvc.LogResult(...)
    }

    // 4. Generate outputs
    reportSvc.Generate()
    evidenceSvc.Finalize()

    return nil
}
```

---

## Testing with Mocks

### Location: `pkg/mocks/registry_service_mock.go`

```go
type MockRegistryService struct {
    ReadValueFunc func(ctx, rootKey, path, valueName) (string, error)
}

func (m *MockRegistryService) ReadValue(
    ctx context.Context,
    rootKey registry.Key,
    path, valueName string,
) (string, error) {
    if m.ReadValueFunc != nil {
        return m.ReadValueFunc(ctx, rootKey, path, valueName)
    }
    return "", nil
}
```

### Unit Test Example

```go
func TestReportRunner_ExecuteReport(t *testing.T) {
    // Create mocks
    mockRegistry := &mocks.MockRegistryService{
        ReadValueFunc: func(ctx, rootKey, path, valueName) (string, error) {
            return "1", nil // Simulate UAC enabled
        },
    }

    // Create test dependencies
    deps := &Dependencies{
        Logger:          slog.Default(),
        Config:          &AppConfig{
            OutputDir:   t.TempDir(),
            EvidenceDir: t.TempDir(),
            ReportsDir:  "testdata",
        },
        RegistryService: mockRegistry, // ← Mock injected
        UIService:       &mocks.MockUIService{},
        ConfigService:   &mocks.MockConfigService{},
        FileService:     &mocks.MockFileService{},
    }

    // Test the runner
    runner := NewReportRunner(deps)
    err := runner.ExecuteReport("test_config.json", true)

    // Assert
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
}
```

---

## Migration Checklist

### Phase 1: Interfaces & Mocks (Week 1)
- [ ] Create `pkg/interfaces.go` with all 6 interfaces
- [ ] Create `pkg/mocks/` directory
- [ ] Implement mock for each interface
- [ ] Write interface compliance tests
- [ ] Verify existing code implements interfaces

### Phase 2: Services (Week 2)
- [ ] Create `pkg/config_service.go`
- [ ] Create `pkg/file_service.go`
- [ ] Update `pkg/htmlreport.go` constructor (add logger, registryService params)
- [ ] Update `pkg/evidence.go` constructor (add logger param)
- [ ] Remove `SetRegistryReader()` method from HTMLReport
- [ ] Write unit tests for all new services

### Phase 3: Dependency Container (Week 3)
- [ ] Create `cmd/dependencies.go` with Dependencies struct
- [ ] Create `cmd/factory.go` with ServiceFactory
- [ ] Update `cmd/toolkit.go` main() to use NewDependencies()
- [ ] Remove `slog.SetDefault(logger)` (global state)
- [ ] Update App struct (remove old fields, add deps + factory)
- [ ] Remove `App.init()` method

### Phase 4: Report Runner (Week 4)
- [ ] Refactor `cmd/report_runner.go` to use ReportRunner struct
- [ ] Extract helper functions (handleError, handleSuccess, etc.)
- [ ] Remove `executeReportQuiet()` duplication
- [ ] Write comprehensive unit tests with mocks
- [ ] Write integration tests

### Phase 5: Integration (Week 5)
- [ ] Update all App methods to use `app.deps` instead of direct fields
- [ ] Update `runReports()`, `viewHTMLReports()`, etc.
- [ ] Integration testing on Windows with real registry
- [ ] Regression testing (run all existing reports)
- [ ] Performance benchmarking (compare before/after)

### Phase 6: Documentation & Polish (Week 6)
- [ ] Update `CLAUDE.md` with DI patterns
- [ ] Update `ARCHITECTURE.md` with new diagrams
- [ ] Create testing guide (`docs/testing/GUIDE.md`)
- [ ] Code review with team
- [ ] Final adjustments based on feedback
- [ ] Merge to main branch

---

## Common Patterns

### Creating a New Service

```go
// 1. Define interface in pkg/interfaces.go
type MyService interface {
    DoSomething() error
}

// 2. Create implementation in pkg/my_service.go
type MyServiceImpl struct {
    logger *slog.Logger
    config *Config
}

func NewMyService(logger *slog.Logger, config *Config) MyService {
    return &MyServiceImpl{
        logger: logger,
        config: config,
    }
}

func (s *MyServiceImpl) DoSomething() error {
    // Implementation
}

// 3. Add to Dependencies struct
type Dependencies struct {
    // ... existing fields
    MyService MyService
}

// 4. Wire in NewDependencies()
func NewDependencies(...) *Dependencies {
    return &Dependencies{
        // ... existing fields
        MyService: pkg.NewMyService(logger, config),
    }
}
```

### Using Dependencies in a Method

```go
// BEFORE ❌
func (app *App) someMethod() {
    reader := pkg.NewRegistryReader(...) // Direct creation
    value, _ := reader.ReadString(...)
}

// AFTER ✅
func (app *App) someMethod() {
    value, _ := app.deps.RegistryService.ReadString(...) // Use injected
}
```

### Testing with Dependency Injection

```go
func TestSomeFeature(t *testing.T) {
    // 1. Create mock
    mock := &mocks.MockRegistryService{
        ReadStringFunc: func(...) (string, error) {
            return "test value", nil
        },
    }

    // 2. Create test dependencies
    deps := &Dependencies{
        RegistryService: mock,
        // ... other deps
    }

    // 3. Test your code
    app := NewApp(deps)
    result := app.someMethod()

    // 4. Assert
    assert.Equal(t, "expected", result)
}
```

---

## Benefits Summary

| Aspect | Before DI | After DI |
|--------|-----------|----------|
| **Testability** | Cannot mock dependencies | Full mock support |
| **Coupling** | High (direct instantiation) | Low (interface-based) |
| **Test Coverage** | ~15% | ~75% |
| **Global State** | 1 (slog.SetDefault) | 0 |
| **Parallel Tests** | ❌ No (global logger) | ✅ Yes |
| **Code Duplication** | 2 executeReport methods | 1 ExecuteReport method |
| **Cyclomatic Complexity** | 45 | 25 |
| **Lines of Code (cmd/)** | ~1000 | ~800 |

---

## Key Principles

1. **Depend on abstractions**: Use interfaces, not concrete types
2. **Constructor injection**: Pass all dependencies via constructors
3. **No global state**: No `slog.SetDefault()`, no package-level vars
4. **Single Responsibility**: Each service does one thing
5. **Factory pattern**: Use factories to create complex objects
6. **Validate early**: Check dependencies in constructors
7. **Explicit dependencies**: All deps visible in function signatures

---

## Rollback Plan

If issues arise during migration:

```bash
# Tag current working version
git tag v1.0.0-before-di

# Create refactoring branch
git checkout -b refactor/dependency-injection

# If rollback needed
git checkout main
git revert <problematic-commits>
# OR
git checkout v1.0.0-before-di
git checkout -b rollback-safe
```

---

## Resources

- **Full Refactoring Plan**: `docs/refactoring/DEPENDENCY_INJECTION_REFACTORING.md`
- **Architecture Diagrams**: `docs/refactoring/DI_ARCHITECTURE_DIAGRAM.md`
- **Testing Guide**: `docs/testing/GUIDE.md` (to be created)
- **SOLID Principles**: https://en.wikipedia.org/wiki/SOLID
- **Go Dependency Injection**: https://blog.drewolson.org/dependency-injection-in-go

---

## FAQ

**Q: Won't DI add a lot of boilerplate?**
A: Initial setup is more code, but reduces duplication and makes testing trivial. Net win.

**Q: Is there a performance overhead?**
A: Negligible. Interface calls in Go are very fast (~1ns overhead).

**Q: Do we need a DI framework?**
A: No. Manual DI (as described here) is simpler and more explicit for this codebase size.

**Q: Can we do this incrementally?**
A: Yes! The 6-week plan is designed for phased migration with testing at each step.

**Q: What if tests fail on non-Windows?**
A: Integration tests with real RegistryReader will be Windows-only. Unit tests with mocks work everywhere.

---

## Contact

For questions about this refactoring:
- Review full plan in `docs/refactoring/DEPENDENCY_INJECTION_REFACTORING.md`
- Check architecture diagrams in `docs/refactoring/DI_ARCHITECTURE_DIAGRAM.md`
- Ask in team discussions before starting a phase
