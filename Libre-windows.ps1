param(
    [Parameter(Position = 0)]
    [ValidateSet("install", "start-bg", "stop", "status")]
    [string]$Command
)

$ErrorActionPreference = "Stop"
$PrefixDir = if ($env:PREFIX_DIR) { $env:PREFIX_DIR } else { Join-Path $HOME ".libretranslate" }
$RepoDir = Join-Path $PrefixDir "LibreTranslate"
$VenvDir = Join-Path $PrefixDir "venv"
$DbDir = Join-Path $PrefixDir "db"
$PidFile = Join-Path $PrefixDir "libretranslate.pid"
$LogFile = Join-Path $PrefixDir "libretranslate.log"
$Python = Join-Path $VenvDir "Scripts\python.exe"
$LibreTranslate = Join-Path $VenvDir "Scripts\libretranslate.exe"
$HostAddress = if ($env:HOST) { $env:HOST } else { "127.0.0.1" }
$Port = if ($env:PORT) { $env:PORT } else { "5100" }
$Threads = if ($env:THREADS) { $env:THREADS } else { "8" }
$RequestLimit = if ($env:REQ_LIMIT) { $env:REQ_LIMIT } else { "60" }

function Initialize-Directories {
    New-Item -ItemType Directory -Force -Path $PrefixDir, $DbDir | Out-Null
}

function Find-SystemPython {
    foreach ($candidate in @("py", "python", "python3")) {
        if (Get-Command $candidate -ErrorAction SilentlyContinue) {
            return $candidate
        }
    }
    throw "Python 3 is required. Install Python 3.11 or newer and run install again."
}

function Install-LibreTranslate {
    Initialize-Directories
    if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
        throw "Git is required. Install Git for Windows and run install again."
    }
    $systemPython = Find-SystemPython
    if (-not (Test-Path (Join-Path $RepoDir ".git"))) {
        & git clone https://github.com/LibreTranslate/LibreTranslate.git $RepoDir
    } else {
        & git -C $RepoDir pull --ff-only
    }
    if ($systemPython -eq "py") {
        & py -3 -m venv $VenvDir
    } else {
        & $systemPython -m venv $VenvDir
    }
    & $Python -m pip install --upgrade pip setuptools wheel
    & $Python -m pip install -e $RepoDir
    & $Python -m pip install argostranslate argos-translate-files translatehtml
    $modelInstaller = @'
from argostranslate import package
package.update_package_index()
available = package.get_available_packages()
for source, target in (("en", "es"), ("es", "en"), ("en", "fr"), ("fr", "en"), ("en", "de"), ("de", "en"), ("en", "it"), ("it", "en"), ("en", "pt"), ("pt", "en"), ("en", "nl"), ("nl", "en")):
    candidate = next((item for item in available if item.from_code == source and item.to_code == target), None)
    if candidate:
        try:
            package.install_from_path(candidate.download())
            print(f"Installed {source}->{target}")
        except Exception as error:
            print(f"Could not install {source}->{target}: {error}")
'@
    $modelInstaller | & $Python -
    Write-Host "LibreTranslate installation completed."
}

function Start-LibreTranslate {
    Initialize-Directories
    if (-not (Test-Path $LibreTranslate)) {
        throw "LibreTranslate is not installed. Run install first."
    }
    if (Test-Path $PidFile) {
        $savedPid = Get-Content $PidFile -ErrorAction SilentlyContinue
        if ($savedPid -and (Get-Process -Id $savedPid -ErrorAction SilentlyContinue)) {
            Write-Host "Already running (PID $savedPid)."
            return
        }
        Remove-Item $PidFile -Force
    }
    $arguments = @("--host", $HostAddress, "--port", $Port, "--threads", $Threads, "--req-limit", $RequestLimit, "--req-limit-storage", "memory://")
    $process = Start-Process -FilePath $LibreTranslate -ArgumentList $arguments -RedirectStandardOutput $LogFile -RedirectStandardError (Join-Path $PrefixDir "libretranslate-error.log") -WindowStyle Hidden -PassThru
    Set-Content -Path $PidFile -Value $process.Id
    Write-Host "Started (PID $($process.Id)) at http://${HostAddress}:$Port."
}

function Stop-LibreTranslate {
    if (-not (Test-Path $PidFile)) {
        Write-Host "Not running."
        return
    }
    $savedPid = Get-Content $PidFile -ErrorAction SilentlyContinue
    if ($savedPid -and (Get-Process -Id $savedPid -ErrorAction SilentlyContinue)) {
        Stop-Process -Id $savedPid -Force
    }
    Remove-Item $PidFile -Force
    Write-Host "Stopped."
}

function Show-Status {
    if (Test-Path $PidFile) {
        $savedPid = Get-Content $PidFile -ErrorAction SilentlyContinue
        if ($savedPid -and (Get-Process -Id $savedPid -ErrorAction SilentlyContinue)) {
            Write-Host "Running (PID $savedPid) at http://${HostAddress}:$Port."
            return
        }
    }
    Write-Host "Not running."
}

switch ($Command) {
    "install" { Install-LibreTranslate }
    "start-bg" { Start-LibreTranslate }
    "stop" { Stop-LibreTranslate }
    "status" { Show-Status }
    default { throw "Usage: .\Libre-windows.ps1 <install|start-bg|stop|status>" }
}
