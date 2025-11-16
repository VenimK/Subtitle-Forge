#requires -Version 5.1
# Subtitle Forge unified dependency installer for Windows (winget + pip)
# - Installs: MKVToolNix, FFmpeg, Deno, Tesseract, Git, Python
# - OCR path: pgsrip via pip (chosen by user)
# - Tessdata: sets up %USERPROFILE%\tessdata_best with common languages
# - Report: writes installation report with versions/paths

$ErrorActionPreference = 'Stop'

# Parameters
[CmdletBinding()]
param(
  [switch]$DryRun
)

$ReportFile = Join-Path $env:USERPROFILE 'subtitle_forge_install_report.txt'
$TessdataDir = Join-Path $env:USERPROFILE 'tessdata_best'
$CommonLangs = @('eng','fra','deu','spa','ita','nld')

function Write-Info($msg){ Write-Host "[INFO] $msg" -ForegroundColor Cyan }
function Write-Warn($msg){ Write-Host "[WARN] $msg" -ForegroundColor Yellow }
function Write-Err($msg){ Write-Host "[ERROR] $msg" -ForegroundColor Red }

function Ensure-Winget {
  if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
    Write-Err 'winget not found. Please install App Installer from Microsoft Store, then re-run.'
    throw 'winget missing'
  }
}

function Install-Packages {
  Ensure-Winget
  Write-Info 'Installing packages via winget...'
  $packages = @(
    'MoritzBunkus.MKVToolNix',
    'Gyan.FFmpeg',        # fallback to FFmpeg.FFmpeg if this fails
    'DenoLand.Deno',
    'Tesseract-OCR.Tesseract-OCR',
    'Git.Git',
    'Python.Python.3'
  )
  foreach($pkg in $packages){
    try {
      if ($DryRun) {
        Write-Host "[DRY] winget install --id $pkg --silent --accept-package-agreements --accept-source-agreements -e"
      } else {
        winget install --id $pkg --silent --accept-package-agreements --accept-source-agreements -e
      }
    }
    catch { Write-Warn "Failed to install $pkg: $_" }
  }
  # FFmpeg fallback
  if (-not (Get-Command ffmpeg -ErrorAction SilentlyContinue)) {
    try {
      if ($DryRun) {
        Write-Host "[DRY] winget install --id 'FFmpeg.FFmpeg' --silent --accept-package-agreements --accept-source-agreements -e"
      } else {
        winget install --id 'FFmpeg.FFmpeg' --silent --accept-package-agreements --accept-source-agreements -e
      }
    }
    catch { Write-Warn "FFmpeg fallback install failed: $_" }
  }
}

function Install-Pgsrip {
  Write-Info 'Installing pgsrip via pip...'
  $py = (Get-Command py -ErrorAction SilentlyContinue)
  if ($py) {
    if ($DryRun) { Write-Host "[DRY] py -m pip install --upgrade pip" } else { py -m pip install --upgrade pip }
    try {
      if ($DryRun) { Write-Host "[DRY] py -m pip install --break-system-packages --ignore-installed pgsrip" }
      else { py -m pip install --break-system-packages --ignore-installed pgsrip }
    } catch {
      if ($DryRun) { Write-Host "[DRY] py -m pip install pgsrip" } else { py -m pip install pgsrip }
    }
  } elseif (Get-Command python -ErrorAction SilentlyContinue) {
    if ($DryRun) { Write-Host "[DRY] python -m pip install --upgrade pip" } else { python -m pip install --upgrade pip }
    try {
      if ($DryRun) { Write-Host "[DRY] python -m pip install --break-system-packages --ignore-installed pgsrip" }
      else { python -m pip install --break-system-packages --ignore-installed pgsrip }
    } catch {
      if ($DryRun) { Write-Host "[DRY] python -m pip install pgsrip" } else { python -m pip install pgsrip }
    }
  } else {
    Write-Err 'Python not available after winget install.'
  }
}

function Setup-Tessdata {
  Write-Info "Setting up tessdata at $TessdataDir"
  if ($DryRun) { Write-Host "[DRY] New-Item -ItemType Directory -Force -Path $TessdataDir" }
  else { New-Item -ItemType Directory -Force -Path $TessdataDir | Out-Null }
  foreach($lang in $CommonLangs){
    $dst = Join-Path $TessdataDir "$lang.traineddata"
    if (-not (Test-Path $dst)){
      $url = "https://github.com/tesseract-ocr/tessdata_best/raw/main/$lang.traineddata"
      Write-Info "Downloading $lang.traineddata"
      try {
        if ($DryRun) { Write-Host "[DRY] Invoke-WebRequest -Uri $url -OutFile $dst -UseBasicParsing" }
        else { Invoke-WebRequest -Uri $url -OutFile $dst -UseBasicParsing }
      }
      catch {
        Write-Warn "Failed to download $lang.traineddata: $_"
      }
    }
  }
}

function Write-Report {
  Write-Info "Writing report to $ReportFile"
  $env:TESSDATA_PREFIX = $TessdataDir
  $lines = @()
  $lines += '=== Subtitle Forge Dependency Report (Windows) ==='
  $lines += (Get-Date).ToString('s')
  $lines += "PATH: $($env:PATH)"
  $lines += "TESSDATA_PREFIX: $($env:TESSDATA_PREFIX)"
  $lines += ''
  $lines += '-- Versions --'
  foreach($cmd in 'ffmpeg','mkvmerge','mkvextract','mkvinfo','deno','tesseract','python','py','pgsrip'){
    if ($DryRun) {
      $lines += "$cmd: (skipped in dry-run)"
    } else {
      try { $ver = & $cmd --version 2>$null; if (-not $ver) { $ver = & $cmd -V 2>$null }
        $lines += "$cmd: $((($ver | Select-Object -First 1) -join ''))"
      } catch { $lines += "$cmd: not found" }
    }
  }
  $lines += ''
  $lines += '-- Locations --'
  foreach($cmd in 'ffmpeg','mkvmerge','mkvextract','mkvinfo','deno','tesseract','python','py','pgsrip'){
    if ($DryRun) {
      $lines += "$cmd: (skipped in dry-run)"
    } else {
      try { $loc = (Get-Command $cmd -ErrorAction Stop).Source; $lines += "$cmd: $loc" }
      catch { $lines += "$cmd: not found" }
    }
  }
  if ($DryRun) {
    Write-Host "[DRY] Set-Content -Encoding UTF8 -Path $ReportFile  (report write skipped)"
  } else {
    $lines | Set-Content -Encoding UTF8 -Path $ReportFile
  }
}

try {
  Install-Packages
  Install-Pgsrip
  Setup-Tessdata
  Write-Report
  Write-Info 'All done. Please review the report if any tool is missing.'
} catch {
  Write-Err $_
  exit 1
}
