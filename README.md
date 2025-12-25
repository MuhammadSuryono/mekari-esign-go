# Mekari eSign API Client

A Go application with clean architecture for integrating with Mekari eSign API using HMAC-SHA256 or OAuth2 authentication.

## ✨ Features

- ✅ Clean Architecture
- ✅ HMAC-SHA256 & OAuth2 authentication for Mekari API
- ✅ Modular design with Uber FX
- ✅ Configuration via YAML + Environment variables
- ✅ Structured logging (Zap)
- ✅ PostgreSQL & Redis support
- ✅ **Windows Service** with auto-start
- ✅ **Bundled installer** with Redis & PostgreSQL
- ✅ **Auto-update** from GitHub Releases

## 🏗️ Architecture

This project follows Clean Architecture principles:

```
├── cmd/
│   ├── main.go                    # Development entry point
│   └── service/
│       └── main.go                # Windows Service entry point
├── internal/
│   ├── config/                    # Configuration management (Viper)
│   ├── domain/
│   │   ├── entity/                # Business entities
│   │   └── repository/            # Repository interfaces
│   ├── infrastructure/
│   │   ├── database/              # PostgreSQL connection
│   │   ├── redis/                 # Redis connection
│   │   ├── httpclient/            # HTTP client with HMAC/OAuth2
│   │   ├── logger/                # Logging (Zap)
│   │   └── repository/            # Repository implementations
│   ├── usecase/                   # Business logic
│   ├── delivery/
│   │   └── http/
│   │       ├── handler/           # HTTP handlers
│   │       └── router/            # Route definitions
│   ├── service/                   # Windows Service wrapper
│   └── server/                    # Server lifecycle management
├── updater/                       # Auto-update from GitHub
├── installer/                     # Windows installer files
├── config.yml                     # Configuration file
├── go.mod                         # Go modules
├── Makefile                       # Build commands
└── build-windows.ps1              # Windows build script
```

## 🛠️ Tech Stack

- **Web Framework**: [GoFiber](https://gofiber.io/) - Fast HTTP framework
- **Configuration**: [Viper](https://github.com/spf13/viper) - Configuration management
- **Dependency Injection**: [Uber FX](https://github.com/uber-go/fx) - Modular DI container
- **Logging**: [Zap](https://github.com/uber-go/zap) - Structured logging
- **Database**: PostgreSQL
- **Cache**: Redis

---

## 🚀 Quick Start

### Prerequisites

- Go 1.23 or later
- PostgreSQL (or use bundled installer on Windows)
- Redis (or use bundled installer on Windows)
- Make (optional)

### Installation (Development)

1. Clone the repository:
```bash
git clone https://github.com/muhammadsuryono/mekari-esign-go.git
cd mekari-esign-go
```

2. Install dependencies:
```bash
go mod tidy
```

3. Configure the application by copying and editing `config.example.yml`:
```bash
cp config.example.yml config.yml
# Edit config.yml with your settings
```

4. Run the application:
```bash
# Using Make
make dev

# Or directly with Go
go run cmd/main.go
```

### Configuration

```yaml
app:
  name: "mekari-esign"
  port: 8080
  env: "development"
  base_url: "http://localhost:8080"

mekari:
  auth_type: "oauth2"  # "oauth2" or "hmac"
  base_url: "https://sandbox-api.mekari.com"
  auth_url: "https://sandbox-account.mekari.com"
  timeout: 30
  oauth2:
    client_id: "YOUR_CLIENT_ID"
    client_secret: "YOUR_CLIENT_SECRET"
  hmac:
    client_id: "YOUR_HMAC_CLIENT_ID"
    client_secret: "YOUR_HMAC_CLIENT_SECRET"

database:
  driver: "postgres"
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "your_password"
  dbname: "mekari_esign"
  sslmode: "disable"

redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0

logging:
  level: "debug"
  format: "json"
```

---

## 🪟 Windows Installation

### Option 1: Full Installer (Recommended)

Download the latest installer from [GitHub Releases](https://github.com/muhammadsuryono/mekari-esign-go/releases) and run `MekariEsignSetup-x.x.x.exe`.

The installer includes:
- ✅ Mekari E-Sign Service (auto-start)
- ✅ Redis (embedded)
- ✅ PostgreSQL (embedded)
- ✅ Management tools
- ✅ Scheduled auto-updates

### Option 2: Manual Installation

1. Download `mekari-esign-windows-amd64.zip` from releases
2. Extract to your desired location
3. Configure `config.yml`
4. Run as Administrator:

```cmd
# Install as Windows Service
mekari-esign.exe -install

# The service will auto-start on Windows boot
```

### Windows Service Commands

```cmd
# Install service (auto-start enabled)
mekari-esign.exe -install

# Start service
mekari-esign.exe -start

# Stop service
mekari-esign.exe -stop

# Uninstall service
mekari-esign.exe -uninstall

# Run in debug/console mode
mekari-esign.exe -debug

# Check for updates
mekari-esign.exe -update

# Show version
mekari-esign.exe -version
```

### Auto-Update

The service automatically checks for updates daily from GitHub Releases. To manually trigger an update:

```cmd
mekari-esign.exe -update
```

To disable auto-update:
```cmd
schtasks /delete /tn "MekariEsignUpdater" /f
```

---

## 🔨 Building

### Build for Current Platform

```bash
# Development build
make build

# Service build (with version info)
make build-service VERSION=1.0.0
```

### Build for Windows (Cross-compile)

```bash
# Build Windows executable
make build-windows VERSION=1.0.0

# Create release package (ZIP)
make release-windows VERSION=1.0.0
```

### Build Complete Installer (Windows)

On Windows with PowerShell:

```powershell
# Full build with installer
.\build-windows.ps1 -Version "1.0.0"

# Skip downloading dependencies
.\build-windows.ps1 -Version "1.0.0" -SkipDownloads

# Clean build artifacts
.\build-windows.ps1 -Clean
```

Requirements:
- Go 1.23+
- PowerShell 5.1+
- [Inno Setup 6](https://jrsoftware.org/isinfo.php) (for installer)

---

## 📡 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/esign/profile` | Get user profile |
| GET | `/api/v1/esign/documents` | Get documents list |
| POST | `/api/v1/esign/documents/request-sign` | Global Request Sign |

### Example Requests

```bash
# Health check
curl http://localhost:8080/health

# Get profile
curl http://localhost:8080/api/v1/esign/profile

# Get documents
curl "http://localhost:8080/api/v1/esign/documents?page=1&per_page=10"
```

### Global Request Sign

Send a base64 encoded PDF document and request signatures from multiple signers.

**Request:** `POST /api/v1/esign/documents/request-sign`

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `doc` | string | Yes | Base64 encoded PDF document |
| `filename` | string | Yes | Document filename |
| `signers` | array | Yes | Array of signer objects |
| `auto_sign` | boolean | No | Auto sign by document owner |
| `is_sequence` | boolean | No | Enable sequential signing |
| `callback_url` | string | No | Webhook callback URL |
| `expiry_date` | string | No | Document expiry date (YYYY-MM-DD) |

**Example:**

```bash
BASE64_DOC=$(base64 -w 0 document.pdf)

curl -X POST http://localhost:8080/api/v1/esign/documents/request-sign \
  -H "Content-Type: application/json" \
  -d '{
    "doc": "'$BASE64_DOC'",
    "filename": "document.pdf",
    "signers": [
      {
        "name": "John Doe",
        "email": "john@example.com",
        "sign_page": "1",
        "signature_positions": [
          {"x": 100, "y": 200, "page": 1}
        ]
      }
    ]
  }'
```

---

## 🔐 Authentication

### OAuth2

The default authentication method. Supports authorization code flow with automatic token refresh.

### HMAC-SHA256

Alternative authentication for server-to-server integration. The signature is generated from:

1. **Date Header**: Current UTC time in RFC1123 format
2. **Request Line**: `{METHOD} {PATH} HTTP/1.1`

Authorization header format:
```
hmac username="{client_id}", algorithm="hmac-sha256", headers="date request-line", signature="{signature}"
```

---

## 🌍 Environment Variables

Override configuration using environment variables:

| Variable | Description |
|----------|-------------|
| `APP_PORT` | Application port |
| `APP_ENV` | Environment (development/production) |
| `MEKARI_AUTH_TYPE` | Authentication type (oauth2/hmac) |
| `MEKARI_OAUTH2_CLIENT_ID` | OAuth2 Client ID |
| `MEKARI_OAUTH2_CLIENT_SECRET` | OAuth2 Client Secret |
| `MEKARI_HMAC_CLIENT_ID` | HMAC Client ID |
| `MEKARI_HMAC_CLIENT_SECRET` | HMAC Client Secret |
| `MEKARI_BASE_URL` | Mekari API Base URL |
| `DATABASE_HOST` | PostgreSQL host |
| `DATABASE_PORT` | PostgreSQL port |
| `DATABASE_USER` | PostgreSQL user |
| `DATABASE_PASSWORD` | PostgreSQL password |
| `REDIS_HOST` | Redis host |
| `REDIS_PORT` | Redis port |

---

## 📦 Releasing New Version

1. Update version in your code
2. Commit and push changes
3. Create a tag:

```bash
git tag v1.0.1
git push origin v1.0.1
```

GitHub Actions will automatically:
- Build Windows executable
- Download Redis & PostgreSQL
- Build Inno Setup installer
- Create GitHub Release with all assets

---

## 📁 Windows Installation Structure

```
C:\Program Files\MekariEsign\
├── mekari-esign.exe          # Main executable
├── config.yml                # Configuration
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

### Windows Services Created

| Service Name | Display Name | Description |
|--------------|--------------|-------------|
| MekariEsign | Mekari E-Sign Service | Main application |
| MekariRedis | Mekari Redis Server | Redis cache |
| MekariPostgres | Mekari PostgreSQL Server | PostgreSQL database |

---

## 🔧 Makefile Commands

```bash
make build           # Build development version
make build-service   # Build service version
make build-windows   # Build for Windows
make release-windows # Create Windows release package
make run             # Build and run
make dev             # Run in development mode
make test            # Run tests
make clean           # Clean build artifacts
make tidy            # Tidy dependencies
make fmt             # Format code
make lint            # Run linter
make check-update    # Check for updates
make version         # Show version
make help            # Show help
```

---

## 📄 License

MIT License

---

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
