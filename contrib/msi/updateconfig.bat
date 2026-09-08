@echo off
setlocal
set "UQDA_CONFIG_DIR=%~1"
if not defined UQDA_CONFIG_DIR exit /b 1
if not exist "%~dp0uqda.exe" exit /b 1
if not exist "%UQDA_CONFIG_DIR%" mkdir "%UQDA_CONFIG_DIR%"
if not exist "%UQDA_CONFIG_DIR%" exit /b 1

rem LocalSystem and Administrators only. Numeric SIDs work on localized Windows.
icacls "%UQDA_CONFIG_DIR%" /inheritance:r /grant:r "*S-1-5-18:(OI)(CI)F" "*S-1-5-32-544:(OI)(CI)F" >nul
if errorlevel 1 exit /b 1

rem Never replace an existing identity, even if its configuration is invalid.
if exist "%UQDA_CONFIG_DIR%\uqda.conf" goto validate
set "UQDA_NEW_CONFIG=%UQDA_CONFIG_DIR%\uqda.conf.%RANDOM%-%RANDOM%.new"
"%~dp0uqda.exe" -genconf > "%UQDA_NEW_CONFIG%"
if errorlevel 1 goto failed_new
"%~dp0uqda.exe" -useconffile "%UQDA_NEW_CONFIG%" -address >nul
if errorlevel 1 goto failed_new
move /y "%UQDA_NEW_CONFIG%" "%UQDA_CONFIG_DIR%\uqda.conf" >nul
if errorlevel 1 goto failed_new

:validate
"%~dp0uqda.exe" -useconffile "%UQDA_CONFIG_DIR%\uqda.conf" -address >nul 2>"%UQDA_CONFIG_DIR%\install-config-error.log"
if errorlevel 1 (
  echo UQDA configuration could not be loaded. Existing identity was preserved.
  echo Inspect "%UQDA_CONFIG_DIR%\install-config-error.log" before retrying.
  exit /b 1
)
exit /b 0

:failed_new
if exist "%UQDA_NEW_CONFIG%" del "%UQDA_NEW_CONFIG%"
echo UQDA could not generate a valid configuration. Check executable trust and directory permissions.
exit /b 1
