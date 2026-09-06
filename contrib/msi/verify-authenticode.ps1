[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string[]]$Path
)

$ErrorActionPreference = "Stop"

foreach ($item in $Path) {
    $resolved = (Resolve-Path -LiteralPath $item).Path
    $signature = Get-AuthenticodeSignature -LiteralPath $resolved

    if ($signature.Status -ne [System.Management.Automation.SignatureStatus]::Valid) {
        throw "Invalid Authenticode signature on '$resolved': $($signature.Status) $($signature.StatusMessage)"
    }
    if ($null -eq $signature.SignerCertificate) {
        throw "No signer certificate was returned for '$resolved'."
    }
    if ($null -eq $signature.TimeStamperCertificate) {
        throw "No RFC3161 timestamp was returned for '$resolved'."
    }

    Write-Host "Valid Authenticode signature: $resolved"
    Write-Host "  Publisher: $($signature.SignerCertificate.Subject)"
    Write-Host "  Timestamp: $($signature.TimeStamperCertificate.Subject)"
}
