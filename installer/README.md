# Mekari E-Sign Windows Installer

Dokumentasi untuk membuild dan menginstall Mekari E-Sign Service di Windows.

## 📦 Apa yang Termasuk dalam Bundle

- **Mekari E-Sign Service** - Layanan utama untuk integrasi e-sign
- **Redis** (Embedded) - Cache dan session storage
- **PostgreSQL** (Embedded) - Database utama
- **NSSM** - Service manager untuk Redis dan PostgreSQL
- **Auto-Update** - Update otomatis dari GitHub Releases

## 🔧 Build Prerequisites

1. **Go 1.21+** - https://go.dev/dl/
2. **PowerShell 5.1+** - Sudah terinstall di Windows 10+
3. **Inno Setup 6** - https://jrsoftware.org/isinfo.php (untuk build installer)

## 🚀 Cara Build

### Option 1: Menggunakan PowerShell Script (Recommended)

```powershell
# Clone repository
git clone https://github.com/muhammadsuryono/mekari-esign-go.git
cd mekari-esign-go

# Build dengan default settings
.\build-windows.ps1

# Atau dengan custom version dan GitHub config
.\build-windows.ps1 -Version "1.0.1" -GitHubOwner "muhammadsuryono" -GitHubRepo "mekari-esign-go"

# Skip download dependencies (jika sudah ada)
.\build-windows.ps1 -SkipDownloads

# Clean build
.\build-windows.ps1 -Clean
```

### Option 2: Menggunakan Makefile (Cross-platform)

```bash
# Build Windows executable saja
make build-windows VERSION=1.0.0

# Build release package (ZIP)
make release-windows VERSION=1.0.0
```

## 📁 Output Files

Setelah build, file output ada di:

```
dist/
├── MekariEsignSetup-1.0.0.exe      # Full installer
├── mekari-esign-windows-amd64.zip  # Portable package
├── mekari-esign-windows-amd64.zip.sha256  # Checksum
└── RELEASE_NOTES.md                # Template release notes
```

## 💾 Instalasi

### Full Installation (Recommended)

1. Jalankan `MekariEsignSetup-x.x.x.exe` sebagai **Administrator**
2. Ikuti wizard instalasi
3. Pilih komponen yang ingin diinstall:
   - ✅ Mekari E-Sign Service (required)
   - ✅ Redis Server (recommended)
   - ✅ PostgreSQL Server (recommended)
   - ✅ Management Tools
4. Konfigurasi database password
5. Selesai! Service akan otomatis berjalan

### Manual Installation

```cmd
# Extract ZIP
# Edit config.yml sesuai kebutuhan

# Install sebagai Windows Service
mekari-esign.exe -install

# Service akan auto-start saat Windows boot
```

## 🔄 Auto-Update

Installer akan membuat scheduled task yang mengecek update setiap hari jam 3:00 pagi.

### Manual Update Check

```cmd
mekari-esign.exe -update
```

### Disable Auto-Update

```cmd
schtasks /delete /tn "MekariEsignUpdater" /f
```

## 📋 Command Line Options

```
mekari-esign.exe [options]

Options:
  -install    Install sebagai Windows Service
  -uninstall  Uninstall Windows Service
  -start      Start service
  -stop       Stop service
  -debug      Jalankan dalam console/debug mode
  -update     Check dan apply updates dari GitHub
  -version    Tampilkan versi
```

## 🗂️ Struktur Instalasi

```
C:\Program Files\MekariEsign\
├── mekari-esign.exe          # Main executable
├── config.yml                # Konfigurasi
├── redis\                    # Redis portable
├── pgsql\                    # PostgreSQL portable
├── tools\
│   └── nssm.exe              # Service manager
├── scripts\                  # Helper scripts
├── data\
│   ├── postgres\             # PostgreSQL data
│   └── redis\                # Redis data
├── documents\
│   ├── ready\                # Documents ready to send
│   ├── progress\             # Documents in progress
│   └── finish\               # Completed documents
└── logs\                     # Log files
```

## 🔌 Windows Services

Installer akan membuat 3 Windows Services:

| Service Name | Display Name | Description |
|--------------|--------------|-------------|
| MekariEsign | Mekari E-Sign Service | Main application service |
| MekariRedis | Mekari Redis Server | Redis cache server |
| MekariPostgres | Mekari PostgreSQL Server | PostgreSQL database |

Semua service dikonfigurasi untuk:
- ✅ Auto-start saat Windows boot
- ✅ Auto-restart jika crash
- ✅ Run as Local System

## 🛠️ Troubleshooting

### Service tidak mau start

```cmd
# Check service status
sc query MekariEsign

# Check logs
type "C:\Program Files\MekariEsign\logs\*.log"

# Jalankan manual untuk lihat error
cd "C:\Program Files\MekariEsign"
mekari-esign.exe -debug
```

### Database connection failed

1. Pastikan PostgreSQL service running:
   ```cmd
   sc query MekariPostgres
   net start MekariPostgres
   ```

2. Check PostgreSQL logs:
   ```cmd
   type "C:\Program Files\MekariEsign\logs\postgresql*.log"
   ```

### Redis connection failed

1. Pastikan Redis service running:
   ```cmd
   sc query MekariRedis
   net start MekariRedis
   ```

2. Check Redis logs:
   ```cmd
   type "C:\Program Files\MekariEsign\logs\redis*.log"
   ```

## 📝 Release Process

Untuk membuat release baru:

1. Update version di code
2. Commit dan push
3. Create tag:
   ```bash
   git tag v1.0.1
   git push origin v1.0.1
   ```
4. GitHub Actions akan otomatis:
   - Build Windows executable
   - Download Redis & PostgreSQL
   - Build Inno Setup installer
   - Create GitHub Release
   - Upload semua assets

## 🔐 Security Notes

- PostgreSQL hanya listen di localhost (127.0.0.1)
- Redis hanya listen di localhost (127.0.0.1)
- Gunakan password yang kuat untuk database
- Pertimbangkan untuk menggunakan code signing certificate untuk production

## 📄 License

[Your License Here]

