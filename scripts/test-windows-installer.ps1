#Requires -Version 5.1
<#
Exercises the generated Windows installer offline: the release download and
version lookup are replaced by local functions, the installer is run against a
locally built archive, and the installed binary, download URL, and temporary
directory cleanup are checked. On Windows a second run holds a file in the
temporary directory open to confirm a cleanup failure does not fail the install.
#>
[CmdletBinding()]
param(
    [string]$InstallerPath = (Join-Path $PSScriptRoot '..\zSDKs\sdk-cli\scripts\install.ps1')
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$InstallerPath = (Resolve-Path -LiteralPath $InstallerPath).Path
$isWindowsHost = [System.Runtime.InteropServices.RuntimeInformation]::IsOSPlatform([System.Runtime.InteropServices.OSPlatform]::Windows)

function Assert-True {
    param([bool]$Condition, [string]$Message)
    if (-not $Condition) {
        throw "Assertion failed: $Message"
    }
    Write-Host "ok: $Message"
}

function New-InstallerSandbox {
    $root = Join-Path ([System.IO.Path]::GetTempPath()) "installer-test-$([guid]::NewGuid())"
    foreach ($name in 'temp', 'localappdata', 'install', 'payload') {
        New-Item -ItemType Directory -Path (Join-Path $root $name) -Force | Out-Null
    }
    $binary = Join-Path (Join-Path $root 'payload') 'cli.exe'
    Set-Content -LiteralPath $binary -Value "stub binary $([guid]::NewGuid())" -NoNewline -Encoding ascii
    $archive = Join-Path $root 'cli_Windows.zip'
    Compress-Archive -LiteralPath $binary -DestinationPath $archive
    [pscustomobject]@{
        Root       = $root
        Temp       = Join-Path $root 'temp'
        InstallDir = Join-Path $root 'install'
        Binary     = $binary
        Archive    = $archive
    }
}

function Invoke-Installer {
    param($Sandbox)
    $env:TEMP = $Sandbox.Temp
    $env:LOCALAPPDATA = Join-Path $Sandbox.Root 'localappdata'
    $env:CLI_INSTALL_DIR = $Sandbox.InstallDir
    $env:CLI_VERSION = 'v0.0.0-test'
    if (-not $env:PROCESSOR_ARCHITECTURE) {
        $env:PROCESSOR_ARCHITECTURE = 'AMD64'
    }
    $global:InstallerTestArchive = $Sandbox.Archive
    $global:InstallerTestDownloads = @()
    $global:InstallerTestOutput = @()
    & $InstallerPath *>&1 | ForEach-Object {
        $line = "$_"
        $global:InstallerTestOutput += $line
        $line
    }
}

function global:Invoke-RestMethod {
    throw 'the installer must not look up the latest release when CLI_VERSION is set'
}

function global:Invoke-WebRequest {
    param([string]$Uri, [string]$OutFile, [switch]$UseBasicParsing)
    $global:InstallerTestDownloads += $Uri
    Copy-Item -LiteralPath $global:InstallerTestArchive -Destination $OutFile
}

Import-Module Microsoft.PowerShell.Archive

$global:InstallerTestOpenHandle = $null
function global:Expand-Archive {
    param([string]$Path, [string]$DestinationPath, [switch]$Force)
    Microsoft.PowerShell.Archive\Expand-Archive -Path $Path -DestinationPath $DestinationPath -Force:$Force
    if ($global:InstallerTestHoldExtractedBinary) {
        $extracted = Join-Path $DestinationPath 'cli.exe'
        $global:InstallerTestOpenHandle = [System.IO.File]::Open($extracted, 'Open', 'Read', 'Read')
    }
}

$global:InstallerTestOutput = @()
try {
    Write-Host '--- install succeeds and cleans up'
    $global:InstallerTestHoldExtractedBinary = $false
    $sandbox = New-InstallerSandbox
    $output = Invoke-Installer $sandbox
    $installed = Join-Path $sandbox.InstallDir 'cli.exe'
    Assert-True (Test-Path -LiteralPath $installed) 'binary is copied into CLI_INSTALL_DIR'
    Assert-True ((Get-Content -LiteralPath $installed -Raw) -eq (Get-Content -LiteralPath $sandbox.Binary -Raw)) 'installed binary matches the archived one'
    Assert-True ($global:InstallerTestDownloads.Count -eq 1) 'exactly one archive download'
    Assert-True ($global:InstallerTestDownloads[0] -like '*/releases/download/v0.0.0-test/cli_Windows_*.zip') "download URL names the pinned version: $($global:InstallerTestDownloads[0])"
    Assert-True (@(Get-ChildItem -LiteralPath $sandbox.Temp -Force).Count -eq 0) 'temporary directory is removed'
    Assert-True (($output -join "`n") -match 'Installation successful') 'installer reports success'

    if ($isWindowsHost) {
        Write-Host '--- install succeeds when the temporary directory cannot be removed'
        $global:InstallerTestHoldExtractedBinary = $true
        $sandbox = New-InstallerSandbox
        $output = Invoke-Installer $sandbox
        $installed = Join-Path $sandbox.InstallDir 'cli.exe'
        Assert-True (Test-Path -LiteralPath $installed) 'binary is copied despite the held temporary file'
        Assert-True (($output -join "`n") -match 'Warning: could not remove temporary directory') 'installer warns about the leftover temporary directory'
        Assert-True (($output -join "`n") -match 'Installation successful') 'installer still reports success'
        Assert-True (@(Get-ChildItem -LiteralPath $sandbox.Temp -Force).Count -eq 1) 'temporary directory is left in place'
        $global:InstallerTestOpenHandle.Dispose()
    }
    else {
        Write-Host '--- skipping held-file cleanup case: open files can be deleted on this platform'
    }
}
catch {
    Write-Host '--- installer output'
    $global:InstallerTestOutput | ForEach-Object { Write-Host "  $_" }
    throw
}
finally {
    if ($global:InstallerTestOpenHandle) {
        $global:InstallerTestOpenHandle.Dispose()
    }
}
