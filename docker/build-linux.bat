@echo off
REM Build script for compliance-server Linux binary
REM This script cross-compiles the Go application for Linux AMD64

echo =====================================
echo Building compliance-server for Linux
echo =====================================
echo.

REM Navigate to project root (parent of docker directory)
cd /d "%~dp0.."

REM Create bin directory if it doesn't exist
if not exist "docker\bin" mkdir "docker\bin"

REM Clean previous Linux builds (keep .exe for debugging)
if exist "docker\bin\compliance-server" del "docker\bin\compliance-server"

REM Build for Linux AMD64
echo Building Linux AMD64 binary...
set CGO_ENABLED=1&& set GOOS=linux&& set GOARCH=amd64&& go build -ldflags="-s -w" -o docker\bin\compliance-server .\cmd\compliance-server

REM Reset environment variables
set CGO_ENABLED=
set GOOS=
set GOARCH=

REM Verify the binary was created
if exist "docker\bin\compliance-server" (
    echo.
    echo Build successful!
    echo Binary location: docker\bin\compliance-server
    dir "docker\bin\compliance-server" | find "compliance-server"
) else (
    echo.
    echo Build failed - binary not found
    exit /b 1
)

echo.
echo =====================================
echo Build complete!
echo =====================================
echo.
echo Next steps:
echo 1. cd docker
echo 2. docker-compose build
echo 3. docker-compose up -d
echo.
