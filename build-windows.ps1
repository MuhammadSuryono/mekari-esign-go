# =============================================================================
# Mekari E-Sign Windows Installer Builder
# =============================================================================
# This script builds the Windows installer bundle with:
# - Main application executable
# - Embedded Redis
# - Embedded PostgreSQL
# - Inno Setup installer
#
# Requirements:
# - Go 1.21+
# - PowerShell 5.1+
# - Inno Setup 6 (optional, for building installer)
#
# Usage:
#   .\build-windows.ps1                    # Build everything
#   .\build-windows.ps1 -SkipDownloads     # Skip downloading dependencies
#   .\build-windows.ps1 -Version "1.0.1"   # Build with specific version
# =============================================================================

param(
    [string]$Version = "1.0.0",
    [string]$GitHubOwner = "muhammadsuryono",
    [string]$GitHubRepo = "mekari-esign-go",
    [switch]$SkipDownloads,
    [switch]$SkipInstaller,
    [switch]$Clean
)

$ErrorActionPreference = "Stop"

# Configuration
$PROJECT_ROOT = $PSScriptRoot
$BUILD_DIR = Join-Path $PROJECT_ROOT "bin\windows"
$DIST_DIR = Join-Path $PROJECT_ROOT "dist"
$EMBEDDED_DIR = Join-Path $PROJECT_ROOT "embedded"
$TOOLS_DIR = Join-Path $PROJECT_ROOT "tools"

# URLs for dependencies
$REDIS_VERSION = "5.0.14.1"
$REDIS_URL = "https://github.com/tporadowski/redis/releases/download/v${REDIS_VERSION}/Redis-x64-${REDIS_VERSION}.zip"

$PGSQL_VERSION = "15.4-1"
$PGSQL_URL = "https://get.enterprisedb.com/postgresql/postgresql-${PGSQL_VERSION}-windows-x64-binaries.zip"

$NSSM_VERSION = "2.24"
# Using GitHub mirror since nssm.cc is unreliable
$NSSM_URL = "https://github.com/kirillkovalenko/nssm/releases/download/v2.24-101-g897c7ad/nssm-2.24-101-g897c7ad.zip"

# =============================================================================
# Helper Functions
# =============================================================================

function Write-Header {
    param([string]$Message)
    Write-Host ""
    Write-Host "=" * 60 -ForegroundColor Cyan
    Write-Host " $Message" -ForegroundColor Cyan
    Write-Host "=" * 60 -ForegroundColor Cyan
}

function Write-Step {
    param([string]$Step, [string]$Message)
    Write-Host "[$Step] $Message" -ForegroundColor Yellow
}

function Write-Success {
    param([string]$Message)
    Write-Host "[OK] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor DarkYellow
}

function Download-File {
    param(
        [string]$Url,
        [string]$OutFile
    )
    
    Write-Host "  Downloading: $Url"
    
    # Use curl if available (faster), otherwise use Invoke-WebRequest
    if (Get-Command curl.exe -ErrorAction SilentlyContinue) {
        & curl.exe -L -o $OutFile $Url --progress-bar
    } else {
        $ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -Uri $Url -OutFile $OutFile -UseBasicParsing
    }
}

function Extract-Zip {
    param(
        [string]$ZipPath,
        [string]$DestPath
    )
    
    Write-Host "  Extracting to: $DestPath"
    Expand-Archive -Path $ZipPath -DestinationPath $DestPath -Force
}

# =============================================================================
# Clean
# =============================================================================

if ($Clean) {
    Write-Header "Cleaning Build Artifacts"
    
    if (Test-Path $BUILD_DIR) { Remove-Item $BUILD_DIR -Recurse -Force }
    if (Test-Path $DIST_DIR) { Remove-Item $DIST_DIR -Recurse -Force }
    if (Test-Path $EMBEDDED_DIR) { Remove-Item $EMBEDDED_DIR -Recurse -Force }
    if (Test-Path $TOOLS_DIR) { Remove-Item $TOOLS_DIR -Recurse -Force }
    
    Write-Success "Cleaned!"
    exit 0
}

# =============================================================================
# Main Build
# =============================================================================

Write-Header "Mekari E-Sign Windows Installer Builder"
Write-Host "Version: $Version"
Write-Host "GitHub: $GitHubOwner/$GitHubRepo"
Write-Host ""

# Create directories
New-Item -ItemType Directory -Force -Path $BUILD_DIR | Out-Null
New-Item -ItemType Directory -Force -Path $DIST_DIR | Out-Null
New-Item -ItemType Directory -Force -Path $EMBEDDED_DIR | Out-Null
New-Item -ItemType Directory -Force -Path $TOOLS_DIR | Out-Null

# =============================================================================
# Step 1: Build Go Application
# =============================================================================

Write-Step "1/5" "Building Go application for Windows..."

$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

$ldflags = @(
    "-X mekari-esign/updater.Version=$Version",
    "-X mekari-esign/updater.DefaultConfig.Owner=$GitHubOwner",
    "-X mekari-esign/updater.DefaultConfig.Repo=$GitHubRepo",
    "-s",
    "-w"
) -join " "

Push-Location $PROJECT_ROOT
try {
    # Ensure dependencies
    Write-Host "  Running go mod tidy..."
    & go mod tidy
    
    # Build
    Write-Host "  Compiling..."
    & go build -ldflags $ldflags -o "$BUILD_DIR\mekari-esign.exe" .\cmd\service\main.go
    
    if ($LASTEXITCODE -ne 0) {
        throw "Go build failed!"
    }
} finally {
    Pop-Location
}

Write-Success "Build complete: $BUILD_DIR\mekari-esign.exe"

# =============================================================================
# Step 2: Download Redis for Windows
# =============================================================================

if (-not $SkipDownloads) {
    Write-Step "2/5" "Preparing Redis for Windows..."
    
    $REDIS_DIR = Join-Path $EMBEDDED_DIR "redis-win"
    
    if (-not (Test-Path "$REDIS_DIR\redis-server.exe")) {
        $redisZip = Join-Path $env:TEMP "redis-win.zip"
        
        Download-File -Url $REDIS_URL -OutFile $redisZip
        
        # Extract
        $tempExtract = Join-Path $env:TEMP "redis-extract"
        Extract-Zip -ZipPath $redisZip -DestPath $tempExtract
        
        # Move to correct location
        if (Test-Path $REDIS_DIR) { Remove-Item $REDIS_DIR -Recurse -Force }
        Move-Item -Path $tempExtract -Destination $REDIS_DIR
        
        # Cleanup
        Remove-Item $redisZip -Force -ErrorAction SilentlyContinue
        
        Write-Success "Redis downloaded!"
    } else {
        Write-Host "  Redis already exists, skipping download."
    }
} else {
    Write-Step "2/5" "Skipping Redis download..."
}

# =============================================================================
# Step 3: Download PostgreSQL for Windows
# =============================================================================

if (-not $SkipDownloads) {
    Write-Step "3/5" "Preparing PostgreSQL for Windows..."
    
    $PGSQL_DIR = Join-Path $EMBEDDED_DIR "pgsql-portable"
    
    if (-not (Test-Path "$PGSQL_DIR\bin\postgres.exe")) {
        $pgsqlZip = Join-Path $env:TEMP "pgsql-win.zip"
        
        Download-File -Url $PGSQL_URL -OutFile $pgsqlZip
        
        # Extract
        $tempExtract = Join-Path $env:TEMP "pgsql-extract"
        Extract-Zip -ZipPath $pgsqlZip -DestPath $tempExtract
        
        # Move to correct location (PostgreSQL extracts to 'pgsql' subdirectory)
        if (Test-Path $PGSQL_DIR) { Remove-Item $PGSQL_DIR -Recurse -Force }
        $extractedDir = Get-ChildItem $tempExtract -Directory | Select-Object -First 1
        Move-Item -Path $extractedDir.FullName -Destination $PGSQL_DIR
        
        # Cleanup
        Remove-Item $pgsqlZip -Force -ErrorAction SilentlyContinue
        Remove-Item $tempExtract -Recurse -Force -ErrorAction SilentlyContinue
        
        Write-Success "PostgreSQL downloaded!"
    } else {
        Write-Host "  PostgreSQL already exists, skipping download."
    }
} else {
    Write-Step "3/5" "Skipping PostgreSQL download..."
}

# =============================================================================
# Step 4: Download NSSM
# =============================================================================

if (-not $SkipDownloads) {
    Write-Step "4/5" "Preparing NSSM (Service Manager)..."
    
    $NSSM_EXE = Join-Path $TOOLS_DIR "nssm.exe"
    
    if (-not (Test-Path $NSSM_EXE)) {
        $nssmZip = Join-Path $env:TEMP "nssm.zip"
        
        Download-File -Url $NSSM_URL -OutFile $nssmZip
        
        # Extract
        $tempExtract = Join-Path $env:TEMP "nssm-extract"
        Extract-Zip -ZipPath $nssmZip -DestPath $tempExtract
        
        # Copy nssm.exe (try win64 first, then any nssm.exe)
        $nssmExeSource = Get-ChildItem -Path $tempExtract -Recurse -Filter "nssm.exe" | 
                         Where-Object { $_.Directory.Name -eq "win64" } | 
                         Select-Object -First 1
        
        if (-not $nssmExeSource) {
            # Fallback: find any nssm.exe
            $nssmExeSource = Get-ChildItem -Path $tempExtract -Recurse -Filter "nssm.exe" | 
                             Select-Object -First 1
        }
        
        if ($nssmExeSource) {
            Copy-Item $nssmExeSource.FullName -Destination $NSSM_EXE
        } else {
            Write-Warn "NSSM executable not found in archive!"
        }
        
        # Cleanup
        Remove-Item $nssmZip -Force -ErrorAction SilentlyContinue
        Remove-Item $tempExtract -Recurse -Force -ErrorAction SilentlyContinue
        
        Write-Success "NSSM downloaded!"
    } else {
        Write-Host "  NSSM already exists, skipping download."
    }
} else {
    Write-Step "4/5" "Skipping NSSM download..."
}

# =============================================================================
# Step 5: Build Installer
# =============================================================================

if (-not $SkipInstaller) {
    Write-Step "5/5" "Building Inno Setup installer..."
    
    # Find Inno Setup compiler
    $ISCC_PATHS = @(
        "C:\Program Files (x86)\Inno Setup 6\ISCC.exe",
        "C:\Program Files\Inno Setup 6\ISCC.exe",
        "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe",
        "${env:ProgramFiles}\Inno Setup 6\ISCC.exe"
    )
    
    $ISCC = $null
    foreach ($path in $ISCC_PATHS) {
        if (Test-Path $path) {
            $ISCC = $path
            break
        }
    }
    
    if ($ISCC) {
        # Update version in setup.iss
        $setupIss = Join-Path $PROJECT_ROOT "installer\inno-setup\setup.iss"
        $content = Get-Content $setupIss -Raw
        $content = $content -replace '#define MyAppVersion ".*"', "#define MyAppVersion `"$Version`""
        Set-Content $setupIss -Value $content
        
        # Build installer
        & $ISCC $setupIss
        
        if ($LASTEXITCODE -eq 0) {
            $installerPath = Join-Path $DIST_DIR "MekariEsignSetup-$Version.exe"
            Write-Success "Installer created: $installerPath"
        } else {
            Write-Warn "Inno Setup build failed!"
        }
    } else {
        Write-Warn "Inno Setup not found. Please install from: https://jrsoftware.org/isinfo.php"
        Write-Host "  After installing, run: ISCC.exe installer\inno-setup\setup.iss"
    }
} else {
    Write-Step "5/5" "Skipping installer build..."
}

# =============================================================================
# Create Release Package (ZIP)
# =============================================================================

Write-Header "Creating Release Package"

$releaseZip = Join-Path $DIST_DIR "mekari-esign-windows-amd64.zip"
$releaseDir = Join-Path $env:TEMP "mekari-esign-release"

# Create release directory
if (Test-Path $releaseDir) { Remove-Item $releaseDir -Recurse -Force }
New-Item -ItemType Directory -Force -Path $releaseDir | Out-Null

# Copy files
Copy-Item "$BUILD_DIR\mekari-esign.exe" -Destination $releaseDir
Copy-Item (Join-Path $PROJECT_ROOT "config.example.yml") -Destination (Join-Path $releaseDir "config.yml")

# Create ZIP
if (Test-Path $releaseZip) { Remove-Item $releaseZip -Force }
Compress-Archive -Path "$releaseDir\*" -DestinationPath $releaseZip -CompressionLevel Optimal

# Calculate checksum
$hash = Get-FileHash $releaseZip -Algorithm SHA256
$checksumFile = Join-Path $DIST_DIR "mekari-esign-windows-amd64.zip.sha256"
"$($hash.Hash.ToLower())  mekari-esign-windows-amd64.zip" | Set-Content $checksumFile

# Cleanup
Remove-Item $releaseDir -Recurse -Force

Write-Success "Release package: $releaseZip"
Write-Success "Checksum: $checksumFile"

# =============================================================================
# Summary
# =============================================================================

Write-Header "Build Complete!"

Write-Host ""
Write-Host "Output files:" -ForegroundColor White
Write-Host "  - Executable: $BUILD_DIR\mekari-esign.exe"
Write-Host "  - Release ZIP: $releaseZip"
Write-Host "  - Checksum: $checksumFile"

if (Test-Path (Join-Path $DIST_DIR "MekariEsignSetup-$Version.exe")) {
    Write-Host "  - Installer: $(Join-Path $DIST_DIR "MekariEsignSetup-$Version.exe")"
}

Write-Host ""
Write-Host "Next steps:" -ForegroundColor White
Write-Host "  1. Create a GitHub release with tag 'v$Version'"
Write-Host "  2. Upload mekari-esign-windows-amd64.zip as release asset"
Write-Host "  3. Upload MekariEsignSetup-$Version.exe as release asset"
Write-Host ""

# Create GitHub release notes template
$releaseNotes = @"
# Mekari E-Sign v$Version

## Installation

### Option 1: Installer (Recommended)
Download and run `MekariEsignSetup-$Version.exe` for a complete installation including:
- Mekari E-Sign Service
- Redis (embedded)
- PostgreSQL (embedded)
- Auto-start configuration
- Scheduled updates

### Option 2: Manual Installation
1. Download `mekari-esign-windows-amd64.zip`
2. Extract to your desired location
3. Configure `config.yml`
4. Run `mekari-esign.exe -install` as Administrator

## Checksums
```
SHA256: $($hash.Hash.ToLower())  mekari-esign-windows-amd64.zip
```

## Changelog
- [Add your changes here]
"@

$releaseNotesFile = Join-Path $DIST_DIR "RELEASE_NOTES.md"
$releaseNotes | Set-Content $releaseNotesFile

Write-Host "  Release notes template: $releaseNotesFile"
Write-Host ""

