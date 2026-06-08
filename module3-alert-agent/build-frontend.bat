@echo off
echo ============================================
echo  Building DLP Alert Console Frontend...
echo ============================================
cd /d "%~dp0frontend"
call npm install --silent
call npm run build
echo.
echo Frontend built to: internal/router/web/
echo Ready to: go build ./cmd/server
