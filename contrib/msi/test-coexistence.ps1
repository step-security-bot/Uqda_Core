[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$YggMsiPath,
    [Parameter(Mandatory = $true)][string]$MsiPath,
    [Parameter(Mandatory = $true)][string]$UpgradeMsiPath
)

# Destructive installation tests run ONLY on a disposable CI Windows runner.
$ErrorActionPreference = 'Stop'
if ($env:CI -ne 'true' -or $env:RUNNER_OS -ne 'Windows') {
    throw 'Run this test only on a disposable Windows CI runner.'
}
$yggMsi = (Resolve-Path $YggMsiPath).Path
$firstMsi = (Resolve-Path $MsiPath).Path
$nextMsi = (Resolve-Path $UpgradeMsiPath).Path
$uqda = Join-Path $env:ProgramFiles 'UQDA\uqda.exe'
$ctl = Join-Path $env:ProgramFiles 'UQDA\uqdactl.exe'
$ygg = Join-Path $env:ProgramFiles 'Yggdrasil\yggdrasil.exe'
$yggCtl = Join-Path $env:ProgramFiles 'Yggdrasil\yggdrasilctl.exe'
$config = Join-Path $env:ProgramData 'UQDA\uqda.conf'
$installedMsi = $null
$yggInstalled = $false
$probeService = $false

function Invoke-TestMsi($Package, $Operation, $Label, $ExpectFailure = $false) {
    $log = Join-Path $env:RUNNER_TEMP "uqda-$Label.log"
    $p = Start-Process msiexec.exe -ArgumentList @($Operation, "`"$Package`"", '/qn', '/norestart', '/l*v', "`"$log`"") -Wait -PassThru
    if ($ExpectFailure) {
        if ($p.ExitCode -ne 1603) { throw "$Label expected MSI 1603, got $($p.ExitCode)" }
    } elseif ($p.ExitCode -notin @(0, 3010)) {
        throw "$Label failed with MSI exit $($p.ExitCode); see $log"
    }
}

function Read-Node($Client, $Endpoint) {
    $deadline = (Get-Date).AddSeconds(45)
    do {
        try {
            $response = & $Client "-endpoint=$Endpoint" -json getSelf 2>$null
            if ($LASTEXITCODE -eq 0) { return ($response | ConvertFrom-Json) }
        } catch {
            # A newly started service may not have opened admin yet.
        }
        Start-Sleep -Seconds 1
    } while ((Get-Date) -lt $deadline)
    throw "Node at $Endpoint is not responding"
}

function Assert-YggUnchanged {
    if ((Get-FileHash $ygg).Hash -ne $script:yggHash) { throw 'Upstream executable was modified' }
    if ((Get-Service Yggdrasil).Status -ne 'Running') { throw 'Upstream service was stopped' }
    $now = Read-Node $yggCtl 'tcp://localhost:9001'
    if ($now.key -ne $script:yggIdentity) { throw 'Upstream identity changed' }
}

try {
    if (Get-Service UQDA,Yggdrasil -ErrorAction SilentlyContinue) { throw 'Runner must be clean' }
    # A legacy service has no new-family MSI registration. Refuse takeover.
    & sc.exe create UQDA binPath= "$env:SystemRoot\System32\cmd.exe" start= demand
    if ($LASTEXITCODE -ne 0) { throw 'Could not create isolated legacy-service fixture' }
    $probeService = $true
    Invoke-TestMsi $firstMsi '/i' 'legacy-guard' $true
    $guardLog = Get-Content (Join-Path $env:RUNNER_TEMP 'uqda-legacy-guard.log') -Raw
    if ($guardLog -notmatch 'older or manually installed UQDA service') { throw 'Failure was not the migration guard' }
    & sc.exe delete UQDA
    if ($LASTEXITCODE -ne 0) { throw 'Fixture service removal failed' }
    $probeService = $false

    Invoke-TestMsi $yggMsi '/i' 'ygg-install'
    $yggInstalled = $true
    $script:yggHash = (Get-FileHash $ygg).Hash
    $script:yggIdentity = (Read-Node $yggCtl 'tcp://localhost:9001').key

    Invoke-TestMsi $firstMsi '/i' 'install'
    $installedMsi = $firstMsi
    $self = Read-Node $ctl 'tcp://localhost:19001'
    if ($self.build_version -ne '0.0.100') { throw 'Wrong first version' }
    $identity = $self.key
    $configHash = (Get-FileHash $config).Hash
    Assert-YggUnchanged

    $adapters = Get-NetAdapter -IncludeHidden
    $uqdaAdapter = $adapters | Where-Object { $_.InterfaceGuid -eq '{db97c42e-e485-4fbe-a30a-7d63b7409c16}' }
    $yggAdapter = $adapters | Where-Object { $_.InterfaceGuid -eq '{8f59971a-7872-4aa6-b2eb-061fc4e9d0a7}' }
    if (-not $uqdaAdapter -or -not $yggAdapter) { throw 'Expected two distinct virtual adapters' }

    $machinePath = [Environment]::GetEnvironmentVariable('Path', 'Machine') -split ';'
    if (($machinePath | ForEach-Object { $_.TrimEnd('\') }) -notcontains (Split-Path $uqda)) { throw 'UQDA PATH entry missing' }

    Invoke-TestMsi $nextMsi '/i' 'upgrade'
    $installedMsi = $nextMsi
    $upgraded = Read-Node $ctl 'tcp://localhost:19001'
    if ($upgraded.build_version -ne '0.0.101' -or $upgraded.key -ne $identity) { throw 'Upgrade changed identity or failed' }
    if ((Get-FileHash $config).Hash -ne $configHash) { throw 'Upgrade modified existing config' }
    Assert-YggUnchanged

    # Validate that bad configuration is rejected BEFORE StartServices, never replaced.
    Stop-Service UQDA
    $savedConfig = [IO.File]::ReadAllBytes($config)
    try {
        [IO.File]::WriteAllText($config, '{ invalid config', [Text.UTF8Encoding]::new($false))
        $badHash = (Get-FileHash $config).Hash
        Invoke-TestMsi $nextMsi '/fa' 'invalid-config-repair' $true
        if ((Get-FileHash $config).Hash -ne $badHash) { throw 'Invalid existing configuration was replaced' }
        if (-not (Test-Path (Join-Path (Split-Path $config) 'install-config-error.log'))) { throw 'Missing config error diagnostic' }
    } finally {
        [IO.File]::WriteAllBytes($config, $savedConfig)
    }
    Start-Service UQDA
    if ((Read-Node $ctl 'tcp://localhost:19001').key -ne $identity) { throw 'Restored identity differs' }

    Invoke-TestMsi $nextMsi '/x' 'uninstall'
    $installedMsi = $null
    if (Get-Service UQDA -ErrorAction SilentlyContinue) { throw 'UQDA service remains' }
    if (Test-Path (Split-Path $uqda)) { throw 'UQDA binaries remain' }
    if ((Get-FileHash $config).Hash -ne $configHash) { throw 'Uninstall removed/changed identity' }
    Assert-YggUnchanged
    Write-Host 'Coexistence, upgrade, legacy-service guard, invalid-config and uninstall tests passed.'
} finally {
    if ($probeService) { & sc.exe delete UQDA }
    if ($installedMsi) { Invoke-TestMsi $installedMsi '/x' 'cleanup' }
    if ($yggInstalled) { Invoke-TestMsi $yggMsi '/x' 'ygg-cleanup' }
}
