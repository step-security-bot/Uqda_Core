[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$MsiPath,

    [Parameter(Mandatory = $true)]
    [string]$ExpectedVersion,

    [switch]$RequireSignature
)

$ErrorActionPreference = "Stop"
$msi = (Resolve-Path -LiteralPath $MsiPath).Path
$installLog = Join-Path $env:RUNNER_TEMP "uqda-msi-install.log"
$uninstallLog = Join-Path $env:RUNNER_TEMP "uqda-msi-uninstall.log"
$installDirectory = Join-Path $env:ProgramFiles "UQDA"
$configPath = Join-Path $env:ProgramData "UQDA\uqda.conf"
$installed = $false

function Invoke-Msi {
    param([string[]]$Arguments, [string]$Operation)

    $process = Start-Process -FilePath "msiexec.exe" -ArgumentList $Arguments -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        throw "MSI $Operation failed with exit code $($process.ExitCode)."
    }
}

function Wait-UqdaService {
    $deadline = (Get-Date).AddSeconds(45)
    do {
        $service = Get-Service -Name "UQDA" -ErrorAction SilentlyContinue
        if ($null -ne $service -and $service.Status -eq "Running") {
            return
        }
        Start-Sleep -Seconds 1
    } while ((Get-Date) -lt $deadline)

    throw "UQDA service did not reach Running state within 45 seconds."
}

try {
    if ($RequireSignature) {
        & "$PSScriptRoot\verify-authenticode.ps1" -Path $msi
    }

    Invoke-Msi -Operation "installation" -Arguments @(
        "/i", "`"$msi`"", "/qn", "/norestart", "/l*v", "`"$installLog`""
    )
    $installed = $true

    $uqda = Join-Path $installDirectory "uqda.exe"
    $uqdactl = Join-Path $installDirectory "uqdactl.exe"
    foreach ($file in @($uqda, $uqdactl, (Join-Path $installDirectory "wintun.dll"))) {
        if (-not (Test-Path -LiteralPath $file -PathType Leaf)) {
            throw "Required installed file is missing: $file"
        }
    }

    if ($RequireSignature) {
        & "$PSScriptRoot\verify-authenticode.ps1" -Path @($uqda, $uqdactl)
    }

    if (-not (Test-Path -LiteralPath $configPath -PathType Leaf)) {
        throw "The installer did not create $configPath."
    }

    $machinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    $pathEntries = $machinePath -split ";" | ForEach-Object { $_.TrimEnd("\") }
    if ($pathEntries -notcontains $installDirectory.TrimEnd("\")) {
        throw "The installer did not add $installDirectory to the machine PATH."
    }

    Wait-UqdaService

    $versionOutput = (& $uqda -version 2>&1) -join "`n"
    if ($LASTEXITCODE -ne 0 -or $versionOutput -notmatch "Build version:\s+$([regex]::Escape($ExpectedVersion))") {
        throw "Unexpected uqda version output: $versionOutput"
    }

    $deadline = (Get-Date).AddSeconds(30)
    do {
        $selfOutput = (& $uqdactl getSelf 2>&1) -join "`n"
        if ($LASTEXITCODE -eq 0 -and $selfOutput -match "Build version:") {
            break
        }
        Start-Sleep -Seconds 1
    } while ((Get-Date) -lt $deadline)
    if ($LASTEXITCODE -ne 0 -or $selfOutput -notmatch "Build version:") {
        throw "uqdactl could not query the installed service: $selfOutput"
    }

    Write-Host $versionOutput
    Write-Host $selfOutput
    Write-Host "Windows MSI installation smoke test passed."
}
finally {
    if ($installed) {
        Invoke-Msi -Operation "uninstallation" -Arguments @(
            "/x", "`"$msi`"", "/qn", "/norestart", "/l*v", "`"$uninstallLog`""
        )

        if ($null -ne (Get-Service -Name "UQDA" -ErrorAction SilentlyContinue)) {
            throw "UQDA service still exists after uninstall."
        }
        if (Test-Path -LiteralPath $installDirectory) {
            throw "UQDA installation directory still exists after uninstall."
        }

        $machinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
        $pathEntries = $machinePath -split ";" | ForEach-Object { $_.TrimEnd("\") }
        if ($pathEntries -contains $installDirectory.TrimEnd("\")) {
            throw "UQDA installation directory remains in machine PATH after uninstall."
        }
        if (-not (Test-Path -LiteralPath $configPath -PathType Leaf)) {
            throw "Normal MSI uninstall removed the persistent node identity."
        }

        Write-Host "Windows MSI uninstall smoke test passed; node identity was preserved."
    }
}
