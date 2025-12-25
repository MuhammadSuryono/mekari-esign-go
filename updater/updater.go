package updater

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Version is set during build via ldflags
var Version = "dev"

// GitHubConfig holds GitHub repository configuration
type GitHubConfig struct {
	Owner string // GitHub username or organization
	Repo  string // Repository name
}

// DefaultConfig - configure this with your GitHub repo
var DefaultConfig = GitHubConfig{
	Owner: "muhammadsuryono", // Change this
	Repo:  "mekari-esign-go", // Change this
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []GitHubAsset `json:"assets"`
}

// GitHubAsset represents a release asset
type GitHubAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ContentType        string `json:"content_type"`
}

// Updater handles auto-updates from GitHub
type Updater struct {
	config     GitHubConfig
	httpClient *http.Client
}

// NewUpdater creates a new Updater instance
func NewUpdater(config GitHubConfig) *Updater {
	return &Updater{
		config: config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// NewDefaultUpdater creates an Updater with default config
func NewDefaultUpdater() *Updater {
	return NewUpdater(DefaultConfig)
}

// CheckForUpdate checks if a new version is available
func (u *Updater) CheckForUpdate(ctx context.Context) (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", u.config.Owner, u.config.Repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "MekariEsign-Updater")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // No releases available
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	// Skip draft and prerelease
	if release.Draft || release.Prerelease {
		return nil, nil
	}

	// Compare versions
	remoteVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(Version, "v")

	if !isNewerVersion(remoteVersion, currentVersion) {
		return nil, nil // Already up to date
	}

	return &release, nil
}

// GetDownloadAsset finds the appropriate asset for current platform
func (u *Updater) GetDownloadAsset(release *GitHubRelease) *GitHubAsset {
	// Expected asset name format: mekari-esign-{os}-{arch}.zip
	expectedName := fmt.Sprintf("mekari-esign-%s-%s.zip", runtime.GOOS, runtime.GOARCH)
	expectedNameAlt := "mekari-esign-windows-amd64.zip" // Common naming

	for _, asset := range release.Assets {
		if asset.Name == expectedName || asset.Name == expectedNameAlt {
			return &asset
		}
	}

	// Try to find any zip file for windows
	for _, asset := range release.Assets {
		if strings.Contains(strings.ToLower(asset.Name), "windows") && strings.HasSuffix(asset.Name, ".zip") {
			return &asset
		}
	}

	return nil
}

// DownloadUpdate downloads the update to a temp file
func (u *Updater) DownloadUpdate(ctx context.Context, asset *GitHubAsset, progressFn func(downloaded, total int64)) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", asset.BrowserDownloadURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "MekariEsign-Updater")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "mekari-esign-update-*.zip")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Download with progress
	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := tmpFile.Write(buf[:n]); writeErr != nil {
				os.Remove(tmpFile.Name())
				return "", writeErr
			}
			downloaded += int64(n)
			if progressFn != nil {
				progressFn(downloaded, asset.Size)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			os.Remove(tmpFile.Name())
			return "", err
		}
	}

	return tmpFile.Name(), nil
}

// VerifyChecksum verifies the downloaded file (optional - if checksum provided in release body)
func (u *Updater) VerifyChecksum(filePath, expectedChecksum string) error {
	if expectedChecksum == "" {
		return nil // No checksum to verify
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actualChecksum := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actualChecksum, expectedChecksum) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// ApplyUpdate extracts and applies the update
func (u *Updater) ApplyUpdate(zipPath string) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	installDir := filepath.Dir(exePath)
	backupDir := filepath.Join(installDir, ".backup")

	// Create backup directory
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	// Extract update to temp directory
	extractDir, err := os.MkdirTemp("", "mekari-esign-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(extractDir)

	if err := extractZip(zipPath, extractDir); err != nil {
		return fmt.Errorf("failed to extract update: %w", err)
	}

	// On Windows, we can't replace a running executable directly
	// Create update script that will be run after service restart
	updateScript := filepath.Join(installDir, "apply-update.bat")
	newExe := filepath.Join(extractDir, "mekari-esign.exe")
	backupExe := filepath.Join(backupDir, fmt.Sprintf("mekari-esign-%s.exe.bak", Version))

	// Check if new executable exists
	if _, err := os.Stat(newExe); os.IsNotExist(err) {
		// Try looking in subdirectory
		files, _ := os.ReadDir(extractDir)
		for _, f := range files {
			if f.IsDir() {
				subNewExe := filepath.Join(extractDir, f.Name(), "mekari-esign.exe")
				if _, err := os.Stat(subNewExe); err == nil {
					newExe = subNewExe
					break
				}
			}
		}
	}

	script := fmt.Sprintf(`@echo off
echo Applying Mekari E-Sign update...
timeout /t 3 /nobreak > nul

echo Stopping service...
net stop MekariEsign 2>nul

echo Backing up current version...
if exist "%s" move /y "%s" "%s"

echo Installing new version...
copy /y "%s" "%s"

echo Starting service...
net start MekariEsign

echo Update complete!
del "%%~f0"
`, exePath, exePath, backupExe, newExe, exePath)

	if err := os.WriteFile(updateScript, []byte(script), 0755); err != nil {
		return err
	}

	fmt.Printf("Update script created at: %s\n", updateScript)
	fmt.Println("The update will be applied when you run the script or restart the service.")

	return nil
}

// CheckAndUpdate performs a full update check and apply cycle
func CheckAndUpdate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	updater := NewDefaultUpdater()

	fmt.Printf("Current version: %s\n", Version)
	fmt.Println("Checking for updates...")

	release, err := updater.CheckForUpdate(ctx)
	if err != nil {
		return err
	}

	if release == nil {
		fmt.Println("You are running the latest version.")
		return nil
	}

	fmt.Printf("New version available: %s\n", release.TagName)
	fmt.Printf("Release notes:\n%s\n\n", release.Body)

	asset := updater.GetDownloadAsset(release)
	if asset == nil {
		return fmt.Errorf("no compatible download found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("Downloading %s (%d MB)...\n", asset.Name, asset.Size/1024/1024)

	zipPath, err := updater.DownloadUpdate(ctx, asset, func(downloaded, total int64) {
		percent := float64(downloaded) / float64(total) * 100
		fmt.Printf("\rDownloading: %.1f%%", percent)
	})
	if err != nil {
		return err
	}
	defer os.Remove(zipPath)
	fmt.Println()

	fmt.Println("Applying update...")
	if err := updater.ApplyUpdate(zipPath); err != nil {
		return err
	}

	fmt.Println("Update downloaded and staged successfully!")
	return nil
}

// isNewerVersion compares semantic versions
func isNewerVersion(remote, current string) bool {
	// Simple string comparison - works for semantic versions
	// For more robust comparison, use a semver library
	remoteParts := strings.Split(remote, ".")
	currentParts := strings.Split(current, ".")

	for i := 0; i < len(remoteParts) && i < len(currentParts); i++ {
		if remoteParts[i] > currentParts[i] {
			return true
		}
		if remoteParts[i] < currentParts[i] {
			return false
		}
	}

	return len(remoteParts) > len(currentParts)
}

// extractZip extracts a zip file to destination directory
func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(filepath.Clean(fpath), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
