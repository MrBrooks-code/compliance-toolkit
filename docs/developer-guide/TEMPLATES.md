# HTML Template System

**Version:** 1.1.0
**Last Updated:** 2025-01-05

Complete guide to the Compliance Toolkit template system for customizing HTML reports, adding features, and maintaining the codebase.

---

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Architecture](#architecture)
4. [Template Features](#template-features)
5. [Customization Guide](#customization-guide)
6. [Data Structures](#data-structures)
7. [Advanced Topics](#advanced-topics)

---

## Overview

The Compliance Toolkit uses a modern template-based system for generating HTML reports with professional styling using Bulma CSS framework and Chart.js for interactive visualizations.

### What's Included

- **Bulma CSS** - Modern, professional styling framework
- **Chart.js** - Interactive compliance charts
- **Dark Mode** - User-toggleable light/dark themes
- **Print Styles** - Professional PDF generation
- **Search & Filter** - Real-time result filtering
- **Collapsible Details** - Expandable registry details
- **Sortable Tables** - Click headers to sort

### Benefits

1. **Maintainability**: Separate HTML from Go code
2. **Flexibility**: Easy to customize without recompiling logic
3. **Modern UI**: Professional Bulma components
4. **Interactive**: Chart.js visualizations
5. **Accessible**: Dark mode, print styles, responsive design
6. **Performance**: Template caching, client-side operations

---

## Quick Start

### View Reports

Double-click any `.html` file in the `output/reports/` directory.

### Interactive Features

| Feature | How to Use |
|---------|------------|
| **Dark Mode** | Click moon/sun icon (top-right) |
| **Search** | Type in search box to filter results |
| **Sort** | Click table headers to sort by column |
| **Filter** | Use dropdown: All Status / Success Only / Errors Only |
| **Expand Details** | Click arrow button on any row |
| **Print/PDF** | Press `Ctrl+P` |

### Customize in 3 Steps

1. **Edit template files** in `pkg/templates/`
2. **Rebuild**: `go build -o ComplianceToolkit.exe ./cmd/toolkit.go`
3. **Run report** to see changes

---

## Architecture

### Directory Structure

```
pkg/
├── templates/
│   ├── html/
│   │   ├── base.html              # Main HTML template
│   │   └── components/
│   │       ├── header.html        # Report header with metadata
│   │       ├── kpi-cards.html     # Executive KPI dashboard
│   │       ├── chart.html         # Chart.js compliance chart
│   │       └── data-table.html    # Sortable results table
│   └── css/
│       ├── main.css               # Main styles with dark mode
│       └── print.css              # Print-optimized styles
├── htmlreport.go                  # Template engine
└── templatedata.go                # Data structures
```

### Embedded Files

Templates are embedded into the compiled binary using Go's `embed` package:

```go
//go:embed templates/html templates/css
var templateFS embed.FS
```

**Benefits:**
- No external dependencies at runtime
- Single executable deployment
- Offline compliance scanning
- Templates bundled with binary

### Template Loading

```go
// Parse templates once for performance
tmpl, err := template.New("base").
    Funcs(funcMap).
    ParseFS(templateFS,
        "templates/html/base.html",
        "templates/html/components/*.html",
        "templates/css/*.css")
```

---

## Template Features

### 1. Bulma CSS Framework

**Features:**
- Modern, responsive component library
- Clean card-based layouts
- Built-in responsive grid system
- Professional typography
- Extensive color utilities

**Components Used:**
- `box` - Card containers
- `columns` - Responsive grid
- `table` - Data tables
- `tag` - Status badges
- `button` - Interactive controls

### 2. Chart.js Integration

**Donut Chart:**
- Interactive compliance visualization
- Animated chart rendering
- Responsive and touch-friendly
- Theme-aware (updates with dark mode)

**Configuration:**
```javascript
const chart = new Chart(ctx, {
    type: 'doughnut',
    data: {
        labels: ['Passed', 'Failed'],
        datasets: [{
            data: [passedCount, failedCount],
            backgroundColor: ['#48c774', '#f14668']
        }]
    }
});
```

### 3. Dark Mode Toggle

**Features:**
- User preference saved to `localStorage`
- Smooth transitions between themes
- CSS variables for easy theming
- Automatic chart theme updates

**Implementation:**
```javascript
function toggleDarkMode() {
    const html = document.documentElement;
    const currentTheme = html.getAttribute('data-theme');
    const newTheme = currentTheme === 'light' ? 'dark' : 'light';
    html.setAttribute('data-theme', newTheme);
    localStorage.setItem('theme', newTheme);
    updateChartTheme(newTheme);
}
```

### 4. Print Stylesheet

**Optimizations:**
- Hides interactive elements (buttons, search bar)
- Expands all collapsible sections
- Professional page breaks
- Print-friendly colors (black on white)
- Footer on each page

**Usage:**
```html
<style media="print">
    {{template "print.css"}}
</style>
```

### 5. Search and Filter

**Search:**
- Real-time search across all fields
- Searches in: name, description, registry paths, values, errors
- Case-insensitive matching

**Filter:**
- All Status - Show everything
- Success Only - Show passing checks
- Errors Only - Show failed checks

**Implementation:**
```javascript
function filterResults() {
    const searchInput = document.getElementById('searchInput').value.toLowerCase();
    const statusFilter = document.getElementById('statusFilter').value;

    // Process rows in pairs (summary + detail)
    for (let i = 0; i < rows.length; i += 2) {
        const summaryRow = rows[i];
        const detailRow = rows[i + 1];

        // Combined text search
        let combinedText = summaryRow.textContent.toLowerCase();
        if (detailRow) {
            combinedText += ' ' + detailRow.textContent.toLowerCase();
        }

        const matchesSearch = combinedText.includes(searchInput);
        const matchesStatus = statusFilter === 'all' || status.includes(statusFilter);

        // Show/hide both rows together
        const display = (matchesSearch && matchesStatus) ? '' : 'none';
        summaryRow.style.display = display;
        if (detailRow) detailRow.style.display = display;
    }
}
```

### 6. Collapsible Sections

**Features:**
- Expand/collapse individual registry details
- Smooth animations
- State preserved per page
- Click anywhere on row to toggle

**Implementation:**
```javascript
function toggleCollapse(button) {
    const currentRow = button.closest('tr');
    const detailRow = currentRow.nextElementSibling;
    const content = detailRow.querySelector('td');
    const icon = button.querySelector('i');

    if (content.style.display === 'none') {
        content.style.display = 'table-cell';
        icon.classList.remove('fa-chevron-down');
        icon.classList.add('fa-chevron-up');
    } else {
        content.style.display = 'none';
        icon.classList.remove('fa-chevron-up');
        icon.classList.add('fa-chevron-down');
    }
}
```

### 7. Sortable Tables

**Features:**
- Click column headers to sort
- Visual sort indicators (▲/▼)
- Client-side sorting (no server required)
- Works with filtering

**Implementation:**
```javascript
function sortTable(columnIndex) {
    const table = document.getElementById('resultsTable');
    const tbody = table.querySelector('tbody');
    const rows = Array.from(tbody.querySelectorAll('tr'));
    const header = table.querySelectorAll('th')[columnIndex];
    const isAscending = header.classList.contains('sort-asc');

    // Sort rows
    rows.sort((a, b) => {
        const aValue = a.children[columnIndex].textContent.trim();
        const bValue = b.children[columnIndex].textContent.trim();

        if (isAscending) {
            return bValue.localeCompare(aValue);
        } else {
            return aValue.localeCompare(bValue);
        }
    });

    // Update UI
    header.classList.toggle('sort-asc');
    header.classList.toggle('sort-desc');
    rows.forEach(row => tbody.appendChild(row));
}
```

---

## Customization Guide

### Change Colors

Edit CSS variables in `pkg/templates/css/main.css`:

```css
:root {
    --bg-primary: #ffffff;
    --bg-secondary: #f5f5f5;
    --text-color: #363636;
    --primary-color: #3273dc;
    --success-color: #48c774;
    --warning-color: #ffdd57;
    --danger-color: #f14668;
}

[data-theme="dark"] {
    --bg-primary: #1a1a1a;
    --bg-secondary: #2a2a2a;
    --text-color: #f8f8f2;
    /* ... dark mode overrides ... */
}
```

### Change Layout

Edit files in `pkg/templates/html/components/`:

**Header (`header.html`):**
```html
{{define "header"}}
<div class="box has-background-primary-light">
    <div class="mb-4">
        <h1 class="title is-2 mb-2">
            <span class="icon-text">
                <span class="icon has-text-primary">
                    <i class="fas fa-shield-alt"></i>
                </span>
                <span>Compliance Toolkit - {{.Metadata.ReportTitle}}</span>
            </span>
        </h1>
    </div>
</div>
{{end}}
```

**KPI Cards (`kpi-cards.html`):**
```html
{{define "kpi-cards"}}
<div class="column is-one-quarter">
    <div class="box has-text-centered">
        <p class="heading">Compliance Rate</p>
        <p class="title">{{printf "%.0f" .ComplianceRate}}%</p>
    </div>
</div>
<!-- More cards... -->
{{end}}
```

### Add Your Logo

Edit `pkg/templates/html/components/header.html`:

```html
<div class="columns is-vcentered">
    <div class="column is-narrow">
        <figure class="image is-64x64">
            <img src="data:image/png;base64,YOUR_BASE64_IMAGE" alt="Company Logo">
        </figure>
    </div>
    <div class="column">
        <h1 class="title">{{.Metadata.ReportTitle}}</h1>
    </div>
</div>
```

### Add Custom Footer

Edit `pkg/templates/html/base.html`:

```html
<footer class="footer">
    <div class="content has-text-centered">
        <p>
            <strong>Your Company Name</strong> - Compliance Division
        </p>
        <p class="is-size-7">Generated on {{.GeneratedAt.Format "2006-01-02 15:04:05"}}</p>
    </div>
</footer>
```

### Change KPI Icons

Edit `pkg/templates/html/components/kpi-cards.html`:

```html
<p class="heading">
    <span class="icon">
        <i class="fas fa-YOUR-ICON-NAME"></i>
    </span>
    <span>Your Metric</span>
</p>
```

**Icon reference:** https://fontawesome.com/icons

### Add Custom Section

1. **Create new template file:**

```html
<!-- pkg/templates/html/components/custom-section.html -->
{{define "custom-section"}}
<div class="box">
    <h3 class="title is-4">Custom Section</h3>
    <div class="content">
        <p>Your custom content here</p>
        <p>Access data: {{.MachineName}}</p>
        <p>Compliance: {{printf "%.1f" .ComplianceRate}}%</p>
    </div>
</div>
{{end}}
```

2. **Add to base template:**

```html
<!-- pkg/templates/html/base.html -->
<section class="section">
    <div class="container">
        {{template "custom-section" .}}
    </div>
</section>
```

3. **Rebuild:**

```bash
go build -o ComplianceToolkit.exe ./cmd/toolkit.go
```

---

## Data Structures

### ReportData

Main data structure passed to templates:

```go
type ReportData struct {
    Metadata       ReportMetadata    // Report version, compliance info
    GeneratedAt    time.Time         // Timestamp
    MachineName    string            // Hostname
    ComplianceRate float64           // Calculated compliance percentage
    TotalQueries   int               // Total checks
    PassedQueries  int               // Successful checks
    FailedQueries  int               // Failed checks
    Results        []QueryResult     // Individual check results
}
```

### ReportMetadata

Report configuration metadata:

```go
type ReportMetadata struct {
    ReportTitle    string   // Display title
    ReportVersion  string   // Report version
    Author         string   // Report author
    Description    string   // Report description
    Category       string   // Report category
    LastUpdated    string   // Last update date
    Compliance     string   // Compliance framework
    Tags           []string // Tags for categorization
}
```

### QueryResult

Individual registry check result:

```go
type QueryResult struct {
    Name          string              // Query identifier
    Description   string              // Human-readable description
    RootKey       string              // Registry root (HKLM, HKCU)
    Path          string              // Registry path
    ValueName     string              // Registry value name
    Operation     string              // Operation type (read, read_all)
    ExpectedValue string              // Expected value (if applicable)
    Value         string              // Single value result
    Values        map[string]string   // Multiple values (for read_all)
    Error         string              // Error message if failed
}
```

### Template Access

Access data in templates:

```html
<!-- Metadata -->
<h1>{{.Metadata.ReportTitle}}</h1>
<p>Version: {{.Metadata.ReportVersion}}</p>

<!-- Machine Info -->
<p>Machine: {{.MachineName}}</p>
<p>Generated: {{.GeneratedAt.Format "2006-01-02 15:04:05"}}</p>

<!-- Compliance Stats -->
<p>Rate: {{printf "%.1f" .ComplianceRate}}%</p>
<p>Total: {{.TotalQueries}}</p>
<p>Passed: {{.PassedQueries}}</p>
<p>Failed: {{.FailedQueries}}</p>

<!-- Results Loop -->
{{range .Results}}
    <div>
        <h3>{{.Name}}</h3>
        <p>{{.Description}}</p>
        <p>Path: {{.RootKey}}\{{.Path}}</p>
        {{if .Error}}
            <p class="has-text-danger">Error: {{.Error}}</p>
        {{else}}
            <p class="has-text-success">Value: {{.Value}}</p>
        {{end}}
    </div>
{{end}}
```

---

## Advanced Topics

### Template Functions

Custom functions available in templates:

**printf:**
```html
{{printf "%.2f" .ComplianceRate}}  <!-- Formats to 2 decimals -->
```

**formatValue (custom):**
```go
funcMap := template.FuncMap{
    "formatValue": func(v interface{}) string {
        switch val := v.(type) {
        case map[string]string:
            // Format as key=value pairs
            var parts []string
            for k, v := range val {
                parts = append(parts, fmt.Sprintf("%s = %s", k, v))
            }
            return strings.Join(parts, "\n")
        case []string:
            return strings.Join(val, "\n")
        default:
            return fmt.Sprintf("%v", v)
        }
    },
}
```

### Browser Compatibility

**Supported:**
- Chrome/Edge 90+ ✅
- Firefox 88+ ✅
- Safari 14+ ✅
- Mobile browsers ✅

**Features Used:**
- CSS Grid
- CSS Variables
- LocalStorage API
- Canvas (for Chart.js)
- Flexbox

### Performance

**Template Caching:**
- Templates parsed once per report generation
- Embedded in binary for fast access
- No disk I/O during runtime

**Client-Side Operations:**
- All interactive features (search, sort, filter) are client-side JavaScript
- No server required
- Works offline
- Instant response

**File Size:**
- Typical report: 100-200 KB
- Includes all CSS/JS inline
- Chart.js loaded from CDN (with fallback)
- No external image dependencies

### Offline Use

Reports are completely self-contained:

✅ All CSS embedded
✅ All JavaScript embedded (except Chart.js CDN)
✅ No external images
✅ Works without internet (after first Chart.js load)

**For fully offline/air-gapped environments:**

1. Download Chart.js locally
2. Embed in template:

```html
<script>
{{template "chart.min.js"}}
</script>
```

3. Rebuild application

---

## Troubleshooting

### Templates Not Found

**Error:** `failed to parse base template`

**Cause:** Missing template files

**Solution:**
```bash
# Verify files exist
ls pkg/templates/html/base.html
ls pkg/templates/html/components/*.html
ls pkg/templates/css/*.css

# Rebuild
go build -o ComplianceToolkit.exe ./cmd/toolkit.go
```

### Dark Mode Not Working

**Cause:** Browser localStorage disabled

**Solution:**
1. Enable localStorage in browser settings
2. Check browser privacy/security settings
3. Try different browser

### Charts Not Rendering

**Causes:**
- No internet connection (first load requires CDN)
- JavaScript disabled
- Console errors

**Solution:**
1. Check internet connection
2. Enable JavaScript in browser
3. Open browser DevTools (F12) → Console tab
4. Look for errors

### Print Issues

**Causes:**
- Print stylesheet not loaded
- Wrong page size settings

**Solution:**
1. Open browser print preview
2. Check print stylesheet in DevTools
3. Verify page size settings (A4/Letter)
4. Try "Print to PDF" instead of physical printer

### Styling Not Updating

**Cause:** Browser cache or build cache

**Solution:**
```bash
# Clear Go build cache
go clean -cache

# Rebuild
go build -o ComplianceToolkit.exe ./cmd/toolkit.go

# Hard refresh browser (Ctrl+F5)
# Or clear browser cache
```

---

## Examples

### Generate Report Programmatically

```go
package main

import (
    "github.com/yourorg/compliance-toolkit/pkg"
)

func main() {
    // Create report
    report := pkg.NewHTMLReport("NIST 800-171 Compliance", "./output/reports")

    // Set metadata
    report.Metadata = pkg.ReportMetadata{
        ReportTitle:   "NIST 800-171 Security Compliance Report",
        ReportVersion: "2.0.0",
        Author:        "Compliance Toolkit",
        Description:   "NIST 800-171 security controls validation",
        Category:      "Security & Compliance",
        LastUpdated:   "2025-01-05",
        Compliance:    "NIST 800-171 Rev 2",
        Tags:          []string{"Security", "NIST", "CUI"},
    }

    // Add results
    report.AddResult("uac_enabled", "UAC Status", "HKLM", "SOFTWARE\\...", "EnableLUA", "1", nil)
    report.AddResult("firewall", "Firewall Status", "HKLM", "SYSTEM\\...", "EnableFirewall", "", errors.New("not found"))

    // Generate
    err := report.Generate()
    if err != nil {
        log.Fatal(err)
    }
}
```

### View Report

```bash
# Open in default browser
start output/reports/NIST_800-171_Security_Compliance_Report_20250105_143022.html
```

### Print to PDF

1. Open report in browser
2. Press `Ctrl+P`
3. Select "Save as PDF" as printer
4. Choose options:
   - Orientation: Portrait
   - Paper: A4 or Letter
   - Margins: Default
5. Save

---

## Migration from Old System

### Breaking Changes

**None** - The API remains the same:

```go
// Old and new both work:
report := pkg.NewHTMLReport("My Report", "./output")
report.AddResult("check1", "Description", value, err)
report.Generate()
```

### Benefits Over Old System

| Feature | Old System | New System |
|---------|-----------|------------|
| **Styling** | Inline CSS | Bulma framework |
| **Charts** | None | Chart.js donut charts |
| **Dark Mode** | No | Yes ✅ |
| **Print** | Basic | Optimized stylesheet ✅ |
| **Search** | No | Real-time search ✅ |
| **Filter** | No | Status filtering ✅ |
| **Sort** | No | Sortable tables ✅ |
| **Responsive** | Limited | Full responsive ✅ |
| **Maintenance** | HTML in Go | Separate templates ✅ |

---

## Next Steps

- ✅ **Try It Out**: Generate a report and explore features
- ✅ **Customize**: Edit colors and layouts
- ✅ **Add Reports**: See [Adding Reports Guide](ADDING_REPORTS.md)
- ✅ **Architecture**: See [Architecture Overview](ARCHITECTURE.md)

---

*Template System Guide v1.0*
*ComplianceToolkit v1.1.0*
*Last Updated: 2025-01-05*
