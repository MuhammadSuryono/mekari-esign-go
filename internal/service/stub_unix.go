//go:build !windows
// +build !windows

package service

// Stub implementations for non-Windows platforms

// RunService is a no-op on non-Windows platforms
func RunService(isDebug bool, app *Application) {
	// On non-Windows, just run the app normally
	app.Run()
}

// InstallService is a no-op on non-Windows platforms
func InstallService(exePath string) error {
	return nil
}

// UninstallService is a no-op on non-Windows platforms
func UninstallService() error {
	return nil
}

// StartService is a no-op on non-Windows platforms
func StartService() error {
	return nil
}

// StopService is a no-op on non-Windows platforms
func StopService() error {
	return nil
}

// IsWindowsService always returns false on non-Windows platforms
func IsWindowsService() (bool, error) {
	return false, nil
}
