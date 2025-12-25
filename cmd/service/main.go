package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"mekari-esign/internal/service"
	"mekari-esign/updater"
)

func main() {
	// Define command line flags
	install := flag.Bool("install", false, "Install Windows service")
	uninstall := flag.Bool("uninstall", false, "Uninstall Windows service")
	start := flag.Bool("start", false, "Start the service")
	stop := flag.Bool("stop", false, "Stop the service")
	debug := flag.Bool("debug", false, "Run in debug/console mode")
	update := flag.Bool("update", false, "Check and apply updates from GitHub")
	version := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Show version
	if *version {
		fmt.Printf("Mekari E-Sign Service\n")
		fmt.Printf("Version: %s\n", updater.Version)
		os.Exit(0)
	}

	// Handle update
	if *update {
		if err := updater.CheckAndUpdate(); err != nil {
			log.Printf("Update failed: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Get executable path
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	// Change to executable directory for config loading
	if err := os.Chdir(filepath.Dir(exePath)); err != nil {
		log.Printf("Warning: could not change to executable directory: %v", err)
	}

	switch {
	case *install:
		err = service.InstallService(exePath)
		if err != nil {
			log.Fatalf("Failed to install service: %v", err)
		}
		fmt.Println("Service installed successfully")

		// Start the service after installation
		err = service.StartService()
		if err != nil {
			log.Printf("Warning: Failed to start service: %v", err)
			fmt.Println("You may need to start the service manually")
		} else {
			fmt.Println("Service started")
		}

	case *uninstall:
		// Try to stop service first
		_ = service.StopService()

		err = service.UninstallService()
		if err != nil {
			log.Fatalf("Failed to uninstall service: %v", err)
		}
		fmt.Println("Service uninstalled successfully")

	case *start:
		err = service.StartService()
		if err != nil {
			log.Fatalf("Failed to start service: %v", err)
		}
		fmt.Println("Service started")

	case *stop:
		err = service.StopService()
		if err != nil {
			log.Fatalf("Failed to stop service: %v", err)
		}
		fmt.Println("Service stopped")

	default:
		// Check if running as Windows service
		isService, err := service.IsWindowsService()
		if err != nil {
			log.Printf("Warning: could not determine if running as service: %v", err)
		}

		app := service.NewApplication()

		if isService {
			// Running as Windows service
			service.RunService(false, app)
		} else if *debug {
			// Running in debug mode
			service.RunService(true, app)
		} else {
			// Running in console mode
			fmt.Println("Mekari E-Sign Service")
			fmt.Printf("Version: %s\n", updater.Version)
			fmt.Println("Running in console mode. Press Ctrl+C to stop.")
			fmt.Println()
			fmt.Println("Available commands:")
			fmt.Println("  -install    Install as Windows service")
			fmt.Println("  -uninstall  Uninstall Windows service")
			fmt.Println("  -start      Start the service")
			fmt.Println("  -stop       Stop the service")
			fmt.Println("  -debug      Run in debug mode")
			fmt.Println("  -update     Check for updates")
			fmt.Println("  -version    Show version")
			fmt.Println()

			app.Run()
		}
	}
}
