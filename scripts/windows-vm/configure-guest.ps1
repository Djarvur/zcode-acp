[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$PublicKeyPath
)

$ErrorActionPreference = "Stop"

$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($identity)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw "Run this script from an elevated PowerShell window"
}

if (-not (Test-Path -LiteralPath $PublicKeyPath)) {
    throw "Public key not found: $PublicKeyPath"
}

$publicKey = (Get-Content -LiteralPath $PublicKeyPath -Raw).Trim()
if ([string]::IsNullOrWhiteSpace($publicKey)) {
    throw "Public key file is empty: $PublicKeyPath"
}

$capability = Get-WindowsCapability -Online -Name "OpenSSH.Server~~~~0.0.1.0"
if ($capability.State -ne "Installed") {
    Add-WindowsCapability -Online -Name "OpenSSH.Server~~~~0.0.1.0"
}

Start-Service sshd
Set-Service -Name sshd -StartupType Automatic

$firewallRule = Get-NetFirewallRule -Name "OpenSSH-Server-In-TCP" -ErrorAction SilentlyContinue
if (-not $firewallRule) {
    New-NetFirewallRule -Name "OpenSSH-Server-In-TCP" -DisplayName "OpenSSH Server (zcode-acp VM)" `
        -Direction Inbound -Protocol TCP -LocalPort 22 -Action Allow -Profile Any
}
else {
    Enable-NetFirewallRule -Name "OpenSSH-Server-In-TCP"
}

$sshDir = Join-Path $env:USERPROFILE ".ssh"
$authorizedKeys = Join-Path $sshDir "authorized_keys"
New-Item -ItemType Directory -Path $sshDir -Force | Out-Null
$keys = @()
if (Test-Path -LiteralPath $authorizedKeys) {
    $keys = @(Get-Content -LiteralPath $authorizedKeys | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
}
if ($keys -notcontains $publicKey) {
    $keys += $publicKey
}
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllLines($authorizedKeys, $keys, $utf8NoBom)

# OpenSSH on Windows rejects authorized_keys when inherited ACLs are too broad.
& icacls.exe $sshDir /inheritance:r /grant:r "$($env:USERNAME):(OI)(CI)F" | Out-Null
& icacls.exe $authorizedKeys /inheritance:r /grant:r "$($env:USERNAME):F" | Out-Null

Restart-Service sshd
Write-Host "OpenSSH Server configured for $env:USERNAME on $env:COMPUTERNAME"
