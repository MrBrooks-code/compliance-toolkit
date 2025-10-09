#!/bin/sh
set -e

# Compliance Server Docker Entrypoint Script
# This script generates server.yaml from environment variables if it doesn't exist

CONFIG_FILE="/app/server.yaml"

# Function to generate config from environment variables
generate_config() {
    cat > "$CONFIG_FILE" <<EOF
# Compliance Toolkit Server Configuration
# Auto-generated from environment variables

# HTTP/HTTPS server settings
server:
  host: "${SERVER_HOST:-0.0.0.0}"
  port: ${SERVER_PORT:-8080}
  tls:
    enabled: ${TLS_ENABLED:-false}
    cert_file: "${TLS_CERT_FILE:-certs/server.crt}"
    key_file: "${TLS_KEY_FILE:-certs/server.key}"

# Database configuration
database:
  type: "${DB_TYPE:-postgres}"
  # PostgreSQL configuration
  host: "${DB_HOST:-postgres}"
  port: ${DB_PORT:-5432}
  name: "${DB_NAME:-compliance}"
  user: "${DB_USER:-compliance}"
  password: "${DB_PASSWORD:-compliance_secure_password}"
  sslmode: "${DB_SSLMODE:-disable}"
  # SQLite configuration (fallback)
  path: "${DB_PATH:-data/compliance.db}"

# Authentication settings - SECURITY CRITICAL
auth:
  enabled: ${AUTH_ENABLED:-true}
  require_key: ${AUTH_REQUIRE_KEY:-true}  # ⚠️ Changed to true for security
  use_hashed_keys: ${USE_HASHED_KEYS:-false}
  api_keys: []
  api_key_hashes: []

  # JWT configuration
  jwt:
    enabled: ${JWT_ENABLED:-true}
    secret_key: "${JWT_SECRET_KEY:-}"
    access_token_lifetime: ${JWT_ACCESS_TOKEN_LIFETIME:-15}
    refresh_token_lifetime: ${JWT_REFRESH_TOKEN_LIFETIME:-7}
    issuer: "${JWT_ISSUER:-ComplianceToolkit}"
    audience: "${JWT_AUDIENCE:-ComplianceToolkit}"

# Web dashboard
dashboard:
  enabled: ${DASHBOARD_ENABLED:-true}
  path: "${DASHBOARD_PATH:-/dashboard}"
  login_message: "${DASHBOARD_LOGIN_MESSAGE:-Welcome to Compliance Toolkit}"

# Logging configuration
logging:
  level: "${LOGGING_LEVEL:-info}"
  format: "${LOGGING_FORMAT:-json}"
  output_path: "${LOGGING_OUTPUT:-stdout}"
EOF

    echo "Generated configuration file: $CONFIG_FILE"
}

# Check if config file exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo "No configuration file found, generating from environment variables..."
    generate_config
else
    echo "Using existing configuration file: $CONFIG_FILE"
fi

# Display configuration info
echo "==================================="
echo "Compliance Server Configuration:"
echo "==================================="
echo "Server Host: ${SERVER_HOST:-0.0.0.0}"
echo "Server Port: ${SERVER_PORT:-8080}"
echo "TLS Enabled: ${TLS_ENABLED:-false}"
echo "Database Type: ${DB_TYPE:-postgres}"
if [ "${DB_TYPE:-postgres}" = "postgres" ]; then
    echo "Database Host: ${DB_HOST:-postgres}"
    echo "Database Port: ${DB_PORT:-5432}"
    echo "Database Name: ${DB_NAME:-compliance}"
    echo "Database User: ${DB_USER:-compliance}"
else
    echo "Database Path: ${DB_PATH:-data/compliance.db}"
fi
echo "Auth Enabled: ${AUTH_ENABLED:-true}"
echo "Dashboard Enabled: ${DASHBOARD_ENABLED:-true}"
echo "Log Level: ${LOGGING_LEVEL:-info}"
echo "==================================="

# Create directories if they don't exist
mkdir -p /app/data /app/logs

# Execute the command
exec "$@"
