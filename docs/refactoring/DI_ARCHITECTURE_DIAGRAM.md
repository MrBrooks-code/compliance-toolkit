# Dependency Injection Architecture Diagrams

## Current Architecture (Before DI)

```
┌────────────────────────────────────────────────────────────────┐
│                           main()                               │
│                       (toolkit.go)                             │
└─────────────────────┬──────────────────────────────────────────┘
                      │
                      │ Creates App struct
                      │
                      ▼
┌────────────────────────────────────────────────────────────────┐
│                          App Struct                            │
├────────────────────────────────────────────────────────────────┤
│  Fields:                                                       │
│  - menu: *Menu (created in main)                               │
│  - reader: *RegistryReader (created in App.init())             │
│  - config: *AppConfig (created in main)                        │
│  - outputDir, logsDir, evidenceDir, reportsDir, exeDir         │
│                                                                │
│  Methods:                                                      │
│  - init() - Creates reader, sets global logger                │
│  - executeReport() - Creates HTMLReport & EvidenceLogger       │
│  - executeReportQuiet() - Duplicate of executeReport()         │
│  - runReports(), viewHTMLReports(), etc.                       │
└───────────┬────────────────────────────────────────────────────┘
            │
            │ executeReport() calls
            │
            ▼
┌────────────────────────────────────────────────────────────────┐
│                     Direct Dependencies                        │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  pkg.LoadConfig(configPath)                                    │
│       ↓                                                        │
│  pkg.NewHTMLReport(reportName, outputDir)                      │
│       ↓                                                        │
│  htmlReport.SetRegistryReader(app.reader)  ← TIGHT COUPLING    │
│       ↓                                                        │
│  pkg.NewEvidenceLogger(evidenceDir, reportType)                │
│       ↓                                                        │
│  evidenceLogger.GatherMachineInfo(app.reader)                  │
│                                                                │
└────────────────────────────────────────────────────────────────┘

                      ⚠️ PROBLEMS ⚠️

┌────────────────────────────────────────────────────────────────┐
│ 1. Global State: slog.SetDefault(logger) in init()            │
│    → Cannot run tests in parallel                             │
│    → Hidden dependency on global logger                        │
│                                                                │
│ 2. Direct Instantiation: Components create their own deps     │
│    → Hard to mock for testing                                 │
│    → Tight coupling between layers                            │
│                                                                │
│ 3. No Interfaces: All concrete types                          │
│    → Cannot swap implementations                              │
│    → Violates Dependency Inversion Principle                  │
│                                                                │
│ 4. Mixed Concerns: App does initialization + business logic   │
│    → Single Responsibility Principle violated                 │
│    → Hard to test business logic in isolation                 │
│                                                                │
│ 5. Code Duplication: executeReport() vs executeReportQuiet()  │
│    → Maintenance nightmare                                    │
│    → Bug fixes must be applied twice                          │
└────────────────────────────────────────────────────────────────┘
```

---

## Target Architecture (After DI)

```
┌──────────────────────────────────────────────────────────────────┐
│                            main()                                │
│                        (toolkit.go)                              │
└────────┬─────────────────────────────────────────────────────────┘
         │
         │ 1. Parse flags & create AppConfig
         │ 2. Setup logger (NO global state)
         │ 3. Create directories
         │
         ▼
┌──────────────────────────────────────────────────────────────────┐
│                    NewDependencies(config, logger)               │
│                      (dependencies.go)                           │
├──────────────────────────────────────────────────────────────────┤
│  Returns Dependencies struct containing:                         │
│                                                                  │
│  • Logger: *slog.Logger                                          │
│  • Config: *AppConfig                                            │
│  • RegistryService: pkg.RegistryService (interface)              │
│  • UIService: pkg.UIService (interface)                          │
│  • ConfigService: pkg.ConfigService (interface)                  │
│  • FileService: pkg.FileService (interface)                      │
│                                                                  │
│  Methods:                                                        │
│  • Validate() - Ensures all deps are non-nil                    │
│  • Clone(config) - Creates copy for testing                     │
└───────────┬──────────────────────────────────────────────────────┘
            │
            │ Dependencies passed to
            │
            ▼
┌──────────────────────────────────────────────────────────────────┐
│                      NewApp(deps)                                │
│                      (toolkit.go)                                │
├──────────────────────────────────────────────────────────────────┤
│  App Struct:                                                     │
│  - deps: *Dependencies                                           │
│  - factory: *ServiceFactory                                      │
│                                                                  │
│  Methods (Business Logic Only):                                 │
│  - runInteractive() - Interactive menu loop                     │
│  - runReports() - Uses deps.UIService, factory                  │
│  - viewHTMLReports() - Uses deps.FileService                    │
│  - listReportsCLI() - Uses deps.FileService                     │
│  - runReportCLI() - Uses ReportRunner                           │
│                                                                  │
│  ✅ NO initialization logic                                     │
│  ✅ NO direct component creation                                │
│  ✅ All dependencies injected                                   │
└───────────┬──────────────────────────────────────────────────────┘
            │
            │ Uses factory to create services
            │
            ▼
┌──────────────────────────────────────────────────────────────────┐
│               ServiceFactory (factory.go)                        │
├──────────────────────────────────────────────────────────────────┤
│  CreateReportService(title, outputDir) → ReportService           │
│       Returns: pkg.NewHTMLReport(title, outputDir, logger, reg)  │
│                                                                  │
│  CreateEvidenceService(dir, type) → EvidenceService              │
│       Returns: pkg.NewEvidenceLogger(dir, type, logger)          │
│                                                                  │
│  CreateReportRunner() → *ReportRunner                            │
│       Returns: NewReportRunner(deps)                             │
└───────────┬──────────────────────────────────────────────────────┘
            │
            │ Factory creates
            │
            ▼
┌──────────────────────────────────────────────────────────────────┐
│                    ReportRunner Service                          │
│                  (report_runner.go)                              │
├──────────────────────────────────────────────────────────────────┤
│  Constructor:                                                    │
│  NewReportRunner(deps *Dependencies) *ReportRunner               │
│                                                                  │
│  Methods:                                                        │
│  ExecuteReport(configFile, quiet) error                          │
│      1. Uses deps.ConfigService.LoadConfig()                    │
│      2. Uses factory.CreateReportService()                      │
│      3. Uses factory.CreateEvidenceService()                    │
│      4. Uses deps.RegistryService for queries                   │
│      5. Generates report and evidence                           │
│                                                                  │
│  ✅ Single responsibility: Execute reports                      │
│  ✅ Testable with mocks                                         │
│  ✅ No code duplication                                         │
└──────────────────────────────────────────────────────────────────┘
```

---

## Interface Layer

```
┌────────────────────────────────────────────────────────────────┐
│                    pkg/interfaces.go                           │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  interface RegistryService {                                   │
│      ReadString(...)                                           │
│      ReadInteger(...)                                          │
│      ReadBinary(...)                                           │
│      ReadStrings(...)                                          │
│      ReadValue(...)                                            │
│      BatchRead(...)                                            │
│  }                                                             │
│                                                                │
│  interface ReportService {                                     │
│      Generate() error                                          │
│      AddResult(...)                                            │
│      AddResultWithDetails(...)                                 │
│      SetMetadata(...)                                          │
│      GetOutputPath() string                                    │
│  }                                                             │
│                                                                │
│  interface EvidenceService {                                   │
│      GatherMachineInfo(RegistryService) error                  │
│      LogResult(...)                                            │
│      Finalize() error                                          │
│      GetSummaryText() string                                   │
│      GetLogPath() string                                       │
│  }                                                             │
│                                                                │
│  interface UIService {                                         │
│      ShowHeader()                                              │
│      ShowMainMenu() int                                        │
│      ShowReportMenuDynamic(...) int                            │
│      ShowError(string)                                         │
│      ShowSuccess(string)                                       │
│      Pause()                                                   │
│      GetIntInput() int                                         │
│      // ...                                                    │
│  }                                                             │
│                                                                │
│  interface ConfigService {                                     │
│      LoadConfig(path) (*Config, error)                         │
│      ParseRootKey(string) (registry.Key, error)                │
│  }                                                             │
│                                                                │
│  interface FileService {                                       │
│      FindReportsDirectory(exeDir) string                       │
│      ResolveDirectory(dir, exeDir) string                      │
│      ListReports(reportsDir) ([]ReportInfo, error)             │
│      OpenBrowser(url) error                                    │
│      OpenFile(path) error                                      │
│  }                                                             │
│                                                                │
└────────────────────────────────────────────────────────────────┘
                                  │
                                  │ Implemented by
                                  │
                    ┌─────────────┴─────────────┐
                    │                           │
                    ▼                           ▼
┌───────────────────────────┐   ┌───────────────────────────┐
│  Production Implementations│   │    Mock Implementations   │
├───────────────────────────┤   ├───────────────────────────┤
│                           │   │                           │
│ RegistryReader            │   │ MockRegistryService       │
│ HTMLReport                │   │ MockReportService         │
│ EvidenceLogger            │   │ MockEvidenceService       │
│ Menu                      │   │ MockUIService             │
│ ConfigServiceImpl         │   │ MockConfigService         │
│ FileServiceImpl           │   │ MockFileService           │
│                           │   │                           │
│ Used in production        │   │ Used in unit tests        │
└───────────────────────────┘   └───────────────────────────┘
```

---

## Dependency Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Application Layers                      │
└─────────────────────────────────────────────────────────────────┘

Layer 1: Entry Point
├─ main() (toolkit.go)
│  └─ Responsibilities:
│     • Parse CLI flags
│     • Create configuration
│     • Setup logger
│     • Wire dependencies
│     • Start application

Layer 2: Dependency Container
├─ Dependencies (dependencies.go)
│  └─ Contains:
│     • Logger (*slog.Logger)
│     • Config (*AppConfig)
│     • All service interfaces
│  └─ Methods:
│     • Validate() - Ensure all deps set
│     • Clone() - For testing

Layer 3: Application Core
├─ App (toolkit.go)
│  └─ Responsibilities:
│     • Interactive menu handling
│     • CLI command routing
│     • User interaction coordination
│  └─ Dependencies:
│     • deps *Dependencies
│     • factory *ServiceFactory

Layer 4: Service Layer
├─ ServiceFactory (factory.go)
│  └─ Creates:
│     • ReportService instances
│     • EvidenceService instances
│     • ReportRunner instances
│
├─ ReportRunner (report_runner.go)
│  └─ Responsibilities:
│     • Execute compliance reports
│     • Coordinate services
│     • Handle errors
│  └─ Dependencies:
│     • deps *Dependencies
│     • factory *ServiceFactory

Layer 5: Business Logic (pkg/)
├─ Services (implement interfaces)
│  ├─ RegistryReader - Registry operations
│  ├─ HTMLReport - Report generation
│  ├─ EvidenceLogger - Audit logging
│  ├─ Menu - User interface
│  ├─ ConfigService - Configuration loading
│  └─ FileService - File operations
│
└─ Domain Models
   ├─ Config, Query, ReportMetadata
   ├─ ReportData, SystemInfo, QueryResult
   └─ ComplianceEvidence, ScanResult

Layer 6: Infrastructure
└─ External Dependencies
   ├─ golang.org/x/sys/windows/registry
   ├─ html/template
   ├─ log/slog
   └─ Standard library

┌─────────────────────────────────────────────────────────────────┐
│                      Dependency Direction                       │
│                                                                 │
│  Higher layers depend on lower layers                           │
│  All layers depend on interfaces (Layer 5 interfaces)           │
│  Lower layers have NO knowledge of higher layers                │
│  Follows Clean Architecture principles                          │
└─────────────────────────────────────────────────────────────────┘
```

---

## Testing Strategy Diagram

```
┌──────────────────────────────────────────────────────────────────┐
│                        Testing Pyramid                           │
└──────────────────────────────────────────────────────────────────┘

                           ▲
                          ╱ ╲
                         ╱   ╲
                        ╱     ╲
                       ╱       ╲
                      ╱    E2E  ╲
                     ╱   Tests   ╲
                    ╱─────────────╲
                   ╱               ╲
                  ╱   Integration   ╲
                 ╱      Tests        ╲
                ╱─────────────────────╲
               ╱                       ╲
              ╱      Unit Tests         ╲
             ╱    (with mocks)           ╲
            ╱─────────────────────────────╲
           ╱_______________________________╲


┌──────────────────────────────────────────────────────────────────┐
│ Unit Tests (70% of tests)                                        │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Test: ReportRunner.ExecuteReport()                              │
│  Mocks: MockRegistryService, MockConfigService,                 │
│         MockReportService, MockEvidenceService                   │
│  Assertion: Correct methods called with correct params          │
│                                                                  │
│  Test: HTMLReport.Generate()                                     │
│  Mocks: MockRegistryService                                      │
│  Assertion: HTML output matches expected template               │
│                                                                  │
│  Test: Dependencies.Validate()                                   │
│  Mocks: None (pure logic)                                        │
│  Assertion: Returns error when deps missing                     │
│                                                                  │
│  ✅ Fast execution (milliseconds)                               │
│  ✅ No external dependencies                                    │
│  ✅ High code coverage (target: 75%+)                           │
│  ✅ Isolated failure detection                                  │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│ Integration Tests (20% of tests)                                │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Test: ReportRunner with Real RegistryReader                    │
│  Real: RegistryReader, FileSystem                               │
│  Mocks: None                                                     │
│  Assertion: Report generated with real registry data            │
│                                                                  │
│  Test: Full Report Generation Flow                              │
│  Real: All services                                              │
│  Mocks: None                                                     │
│  Assertion: HTML + Evidence files created correctly             │
│                                                                  │
│  ⚠️  Requires Windows environment                               │
│  ⚠️  Slower execution (seconds)                                 │
│  ✅ Tests real interactions                                     │
│  ✅ Catches integration issues                                  │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│ End-to-End Tests (10% of tests)                                 │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Test: CLI - List Reports                                       │
│  Command: ./ComplianceToolkit.exe -list                         │
│  Assertion: All reports listed, exit code 0                     │
│                                                                  │
│  Test: CLI - Run Single Report                                  │
│  Command: ./ComplianceToolkit.exe -report=NIST_800_171.json     │
│  Assertion: HTML generated, evidence logged, exit code 0        │
│                                                                  │
│  Test: Interactive Mode - Run All Reports                       │
│  Input: Simulated user input (1 → all reports → 0)              │
│  Assertion: All reports completed, files created                │
│                                                                  │
│  ⚠️  Requires Windows + admin privileges                        │
│  ⚠️  Slowest execution (minutes)                                │
│  ✅ Tests complete user workflows                               │
│  ✅ Smoke tests for releases                                    │
└──────────────────────────────────────────────────────────────────┘
```

---

## Before/After Comparison

```
┌────────────────────────────────────────────────────────────────┐
│                  BEFORE (Current State)                        │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  main()                                                        │
│   └─ app := &App{...}                                          │
│       └─ app.init()                                            │
│           └─ slog.SetDefault(logger) ❌ GLOBAL STATE           │
│           └─ app.reader = pkg.NewRegistryReader(...)           │
│                                                                │
│  app.executeReport()                                           │
│   └─ htmlReport := pkg.NewHTMLReport(...)                      │
│   └─ htmlReport.SetRegistryReader(app.reader) ❌ SETTER        │
│   └─ evidence := pkg.NewEvidenceLogger(...)                    │
│   └─ evidence.GatherMachineInfo(app.reader)                    │
│                                                                │
│  Testing:                                                      │
│   ❌ Cannot mock RegistryReader                                │
│   ❌ Cannot test executeReport() in isolation                  │
│   ❌ Tests interfere with each other (global logger)           │
│   ❌ No interfaces to implement                                │
│                                                                │
│  Metrics:                                                      │
│   • Test Coverage: 15%                                         │
│   • Cyclomatic Complexity: 45                                  │
│   • Coupling: High                                             │
│   • Testability: Low                                           │
└────────────────────────────────────────────────────────────────┘

                             ⬇️  REFACTORING  ⬇️

┌────────────────────────────────────────────────────────────────┐
│                   AFTER (Target State)                         │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  main()                                                        │
│   └─ logger := setupLogger(...) ✅ LOCAL SCOPE                 │
│   └─ deps := NewDependencies(config, logger)                   │
│   └─ app, err := NewApp(deps) ✅ CONSTRUCTOR INJECTION         │
│   └─ app.runInteractive()                                      │
│                                                                │
│  ReportRunner.ExecuteReport()                                  │
│   └─ reportSvc := factory.CreateReportService(...)             │
│   └─ evidenceSvc := factory.CreateEvidenceService(...)         │
│   └─ evidenceSvc.GatherMachineInfo(deps.RegistryService)       │
│   └─ reportSvc.Generate()                                      │
│                                                                │
│  Testing:                                                      │
│   ✅ Mock all interfaces easily                                │
│   ✅ Test any component in isolation                           │
│   ✅ Parallel test execution                                   │
│   ✅ 6 interfaces implemented                                  │
│                                                                │
│  Metrics:                                                      │
│   • Test Coverage: 75%                                         │
│   • Cyclomatic Complexity: 25                                  │
│   • Coupling: Low                                              │
│   • Testability: High                                          │
└────────────────────────────────────────────────────────────────┘
```

---

## Migration Path

```
Week 1: Interfaces & Mocks
┌────────────────────────────────────────────────────────────────┐
│ 1. Create pkg/interfaces.go                                    │
│ 2. Create pkg/mocks/ directory                                 │
│ 3. Write MockRegistryService, MockReportService, etc.          │
│ 4. Write interface compliance tests                            │
│ 5. Verify existing code matches interfaces                     │
└────────────────────────────────────────────────────────────────┘
                              ⬇️
Week 2: Services
┌────────────────────────────────────────────────────────────────┐
│ 1. Create pkg/config_service.go                                │
│ 2. Create pkg/file_service.go                                  │
│ 3. Update pkg/htmlreport.go constructor                        │
│ 4. Update pkg/evidence.go constructor                          │
│ 5. Remove SetRegistryReader() method                           │
│ 6. Write unit tests for all services                           │
└────────────────────────────────────────────────────────────────┘
                              ⬇️
Week 3: Dependency Container
┌────────────────────────────────────────────────────────────────┐
│ 1. Create cmd/dependencies.go                                  │
│ 2. Create cmd/factory.go                                       │
│ 3. Update cmd/toolkit.go main()                                │
│ 4. Remove global logger (slog.SetDefault)                      │
│ 5. Update App struct                                            │
│ 6. Test with real dependencies                                 │
└────────────────────────────────────────────────────────────────┘
                              ⬇️
Week 4: Report Runner
┌────────────────────────────────────────────────────────────────┐
│ 1. Refactor cmd/report_runner.go                               │
│ 2. Extract helper functions                                    │
│ 3. Remove executeReportQuiet() duplication                     │
│ 4. Write comprehensive unit tests                              │
│ 5. Integration tests with mocks                                │
└────────────────────────────────────────────────────────────────┘
                              ⬇️
Week 5: Integration
┌────────────────────────────────────────────────────────────────┐
│ 1. Update all App methods                                      │
│ 2. Remove old initialization code                              │
│ 3. Integration testing (Windows)                               │
│ 4. Regression testing (all reports)                            │
│ 5. Performance benchmarking                                    │
└────────────────────────────────────────────────────────────────┘
                              ⬇️
Week 6: Polish & Documentation
┌────────────────────────────────────────────────────────────────┐
│ 1. Update CLAUDE.md                                            │
│ 2. Update ARCHITECTURE.md                                      │
│ 3. Add testing guide                                            │
│ 4. Code review & final adjustments                             │
│ 5. Merge to main branch                                        │
└────────────────────────────────────────────────────────────────┘
```

---

## Success Criteria

✅ All interfaces defined and documented
✅ All services implement interfaces correctly
✅ Zero global state (no `slog.SetDefault()`)
✅ Dependencies container validates all deps
✅ Factory creates all service instances
✅ Unit test coverage ≥ 75%
✅ All existing reports still work
✅ No performance regression (within 5%)
✅ Code complexity reduced (cyclomatic < 30)
✅ Documentation updated
✅ Rollback plan tested
✅ Team trained on new architecture
