# Settings Page Enhancements

**Date:** October 6, 2025
**Version:** 1.1.0

## Overview

Enhanced the Settings page with live API management and server configuration viewing capabilities. The settings page now dynamically loads and manages server configuration through REST API endpoints.

---

## New Backend API Endpoints

### 1. GET `/api/v1/settings/config`
**Purpose:** Retrieve current server configuration (sanitized)

**Authentication:** Required (Bearer token)

**Response:**
```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8443,
    "tls": {
      "enabled": true,
      "cert_file": "server.crt",
      "key_file": "server.key"
    }
  },
  "database": {
    "type": "sqlite",
    "path": "./data/compliance.db"
  },
  "auth": {
    "enabled": true,
    "require_key": true,
    "key_count": 2
  },
  "dashboard": {
    "enabled": true,
    "path": "/dashboard"
  },
  "logging": {
    "level": "info",
    "format": "json"
  }
}
```

**Security:** API keys are NOT included in response (only key_count shown)

---

### 2. POST `/api/v1/settings/config/update`
**Purpose:** Update server configuration (runtime only)

**Authentication:** Required (Bearer token)

**Request Body:**
```json
{
  "logging": {
    "level": "debug"
  }
}
```

**Response:**
```json
{
  "status": "success",
  "message": "Configuration updated (runtime only)"
}
```

**Limitations:**
- Updates runtime config only (not persisted to YAML file)
- Limited to non-sensitive settings for safety
- Currently only supports logging level updates
- Future: Add YAML file persistence

---

### 3. GET `/api/v1/settings/apikeys`
**Purpose:** List all API keys (masked for security)

**Authentication:** Required (Bearer token)

**Response:**
```json
[
  {
    "id": "key-1",
    "key": "test****2345",
    "masked": "true"
  },
  {
    "id": "key-2",
    "key": "demo****7890",
    "masked": "true"
  }
]
```

**Security:** Keys are masked showing only first 4 and last 4 characters

---

### 4. POST `/api/v1/settings/apikeys/add`
**Purpose:** Add a new API key to the server

**Authentication:** Required (Bearer token)

**Request Body:**
```json
{
  "key": "new-api-key-abc123xyz"
}
```

**Validation:**
- Key cannot be empty
- Checks for duplicate keys
- Minimum recommended length: 16 characters

**Response:**
```json
{
  "status": "success",
  "message": "API key added successfully"
}
```

**Limitations:**
- Adds to runtime config only
- Must update `server.yaml` and restart for persistence

---

### 5. POST `/api/v1/settings/apikeys/delete`
**Purpose:** Remove an API key from the server

**Authentication:** Required (Bearer token)

**Request Body:**
```json
{
  "key": "api-key-to-delete"
}
```

**Response:**
```json
{
  "status": "success",
  "message": "API key deleted successfully"
}
```

**Security Warning:** Clients using deleted keys will immediately lose access

---

## Frontend Enhancements

### Dynamic Server Information Display

**Before:** Hardcoded static values
**After:** Live data loaded from `/api/v1/settings/config`

**Displayed Fields:**
- Server Version (from `/` endpoint)
- Server Address (protocol://host:port)
- TLS Status (Enabled/Disabled badge with color coding)
- Database Type (SQLite/PostgreSQL/MySQL)
- Database Path (file location)
- Authentication Status (shows key count)

**Auto-detection:**
- Green badge = Enabled/Secure
- Yellow badge = Warning/Disabled TLS
- Red badge = Disabled security features

---

### Live API Key Management

**Before:** Static hardcoded list with placeholder functions
**After:** Dynamic list with full CRUD operations

**Features:**
1. **Load Keys:**
   - Fetches from `/api/v1/settings/apikeys`
   - Displays masked keys (first 4 + last 4 characters)
   - Shows "Active" status badge

2. **Add New Key:**
   - Random key generator (32 characters)
   - Minimum length validation (16 chars)
   - Duplicate detection
   - Real-time API call to add key
   - Refreshes list after successful add

3. **Copy Key:**
   - One-click copy to clipboard
   - Success notification

4. **Delete Key:**
   - Confirmation dialog
   - Real-time API call to delete
   - Refreshes list after successful delete
   - Warning about client impact

**User Experience:**
- Loading states
- Error handling with user-friendly messages
- Success/failure notifications
- Auto-refresh after changes

---

## JavaScript Functions Added/Updated

### `loadServerInfo()`
```javascript
async function loadServerInfo()
```
- Fetches server configuration
- Updates server info section dynamically
- Handles TLS, database, and auth status display
- Error handling for failed loads

### `loadApiKeys()`
```javascript
async function loadApiKeys()
```
- Fetches API keys from server
- Renders masked key list
- Handles empty state
- Error handling

### `addApiKey()`
```javascript
async function addApiKey()
```
- **Changed from:** Placeholder alert
- **Changed to:** Real API call to `/api/v1/settings/apikeys/add`
- Validates input
- Calls API
- Refreshes key list on success

### `confirmDeleteKey(key)`
```javascript
async function confirmDeleteKey(key)
```
- **Changed from:** Placeholder alert
- **Changed to:** Real API call to `/api/v1/settings/apikeys/delete`
- Confirmation dialog
- Calls API
- Refreshes key list on success

---

## Security Considerations

### API Key Masking
**Implementation:**
```go
func maskAPIKey(key string) string {
    if len(key) <= 8 {
        return "****"
    }
    return key[:4] + "****" + key[len(key)-4:]
}
```

**Example:**
- Input: `test-api-key-12345`
- Output: `test****2345`

### Sanitized Configuration
- API keys are NEVER sent in config responses
- Only key count is exposed
- Certificate paths shown but not contents
- Database credentials excluded (future PostgreSQL support)

### Authentication Required
- All settings endpoints require Bearer token
- Uses existing `authMiddleware`
- Prevents unauthorized configuration changes

---

## Known Limitations

1. **Runtime Only Updates:**
   - API key changes are runtime only
   - Server restart loses changes
   - Must manually update `server.yaml` for persistence
   - **Future Enhancement:** Add YAML file write capability

2. **Limited Config Updates:**
   - Only logging level can be updated via API
   - Other settings read-only for safety
   - **Future Enhancement:** Expand editable settings with proper validation

3. **No Audit Trail:**
   - Configuration changes not logged to database
   - Only server logs record changes
   - **Future Enhancement:** Audit log page (see FUTURE_ENHANCEMENTS.md)

4. **No Key Expiration:**
   - API keys don't expire
   - No rotation mechanism
   - **Future Enhancement:** JWT tokens with expiration

---

## Testing

### Test API Key Management

```powershell
# Start server
cd cmd/compliance-server
.\compliance-server.exe --config server.yaml

# In browser, navigate to:
https://localhost:8443/settings

# Test operations:
# 1. Verify server info loads correctly
# 2. Click "Add New API Key"
# 3. Click "Generate Random" button
# 4. Save the key
# 5. Verify it appears in the list (masked)
# 6. Click "Copy" to test clipboard
# 7. Click "Delete" and confirm
# 8. Verify it's removed from list
```

### Test via API (PowerShell)

```powershell
# Set variables
$ServerUrl = "https://localhost:8443"
$ApiKey = "demo-key-67890"
$Headers = @{
    "Authorization" = "Bearer $ApiKey"
    "Content-Type" = "application/json"
}

# Get server config
Invoke-RestMethod -Uri "$ServerUrl/api/v1/settings/config" -Headers $Headers -SkipCertificateCheck

# Get API keys
Invoke-RestMethod -Uri "$ServerUrl/api/v1/settings/apikeys" -Headers $Headers -SkipCertificateCheck

# Add new API key
$NewKey = @{ key = "test-new-key-$(Get-Random)" } | ConvertTo-Json
Invoke-RestMethod -Uri "$ServerUrl/api/v1/settings/apikeys/add" `
    -Method POST -Headers $Headers -Body $NewKey -SkipCertificateCheck

# Delete API key
$DeleteKey = @{ key = "test-new-key-12345" } | ConvertTo-Json
Invoke-RestMethod -Uri "$ServerUrl/api/v1/settings/apikeys/delete" `
    -Method POST -Headers $Headers -Body $DeleteKey -SkipCertificateCheck
```

---

## Future Enhancements

See `docs/project/FUTURE_ENHANCEMENTS.md` for complete roadmap.

**Settings Page Related:**

1. **YAML File Persistence** (High Priority)
   - Write changes back to `server.yaml`
   - Atomic file updates
   - Backup old config before changes
   - Validation before applying

2. **API Key Features:**
   - Key expiration dates
   - Key rotation mechanism
   - Usage statistics per key
   - Last used timestamp
   - Key descriptions/labels

3. **Configuration Editor:**
   - Edit server host/port
   - Edit TLS certificate paths
   - Edit database connection
   - Toggle features on/off
   - Validation before applying

4. **Alert Thresholds:**
   - Configure alert rules
   - Set notification channels
   - Email/Slack/Teams integration
   - Alert history viewer

5. **Audit Log Integration:**
   - Track who changed what
   - Configuration change history
   - User login tracking
   - Export audit reports

---

## Files Modified

| File | Changes | Lines Added |
|------|---------|-------------|
| `cmd/compliance-server/server.go` | Added 5 new API endpoints + handlers | ~200 |
| `cmd/compliance-server/settings.html` | Enhanced with live data loading | ~100 |

**Total:** ~300 lines of new code

---

## Commit Message Template

```
feat: Add live API key management to settings page

- Add GET /api/v1/settings/config endpoint
- Add POST /api/v1/settings/config/update endpoint
- Add GET /api/v1/settings/apikeys endpoint
- Add POST /api/v1/settings/apikeys/add endpoint
- Add POST /api/v1/settings/apikeys/delete endpoint
- Implement API key masking for security
- Add dynamic server info loading
- Add loadServerInfo() JavaScript function
- Add loadApiKeys() JavaScript function
- Update addApiKey() to call real API
- Update confirmDeleteKey() to call real API
- Display TLS status with color-coded badges
- Display authentication status with key count
- Add success/error notifications
- Add input validation for API keys

Limitations:
- Runtime only (not persisted to YAML)
- Limited config update support
- No audit trail yet

Related: #4 Settings/Configuration Page (FUTURE_ENHANCEMENTS.md)
```

---

## Summary

The settings page has been significantly enhanced with:

✅ **5 new REST API endpoints** for configuration management
✅ **Live server information** display (version, address, TLS, database, auth)
✅ **Full API key CRUD operations** (list, add, delete with validation)
✅ **Security features** (key masking, authentication required)
✅ **User-friendly** notifications and error handling
✅ **Ready for use** with runtime management capabilities

**Next Steps:**
- Add YAML file persistence for permanent changes
- Add audit logging for configuration changes
- Expand configuration editor capabilities
- Implement alert threshold configuration

---

**Implementation Date:** October 6, 2025
**Status:** Complete and Ready for Testing
