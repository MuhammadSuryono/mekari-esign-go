//go:build windows
// +build windows

package service

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const ServiceName = "MekariEsign"
const ServiceDisplayName = "Mekari E-Sign Service"
const ServiceDescription = "Mekari E-Sign Integration Service for document signing"

var elog debug.Log

// MekariEsignService implements svc.Handler
type MekariEsignService struct {
	app *Application
}

func (s *MekariEsignService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	// Start your application
	go s.app.Run()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	elog.Info(1, fmt.Sprintf("%s service started", ServiceName))

loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				elog.Info(1, fmt.Sprintf("%s service stopping", ServiceName))
				s.app.Shutdown()
				break loop
			default:
				elog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

// RunService runs the service
func RunService(isDebug bool, app *Application) {
	var err error
	if isDebug {
		elog = debug.New(ServiceName)
	} else {
		elog, err = eventlog.Open(ServiceName)
		if err != nil {
			return
		}
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("starting %s service", ServiceName))
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(ServiceName, &MekariEsignService{app: app})
	if err != nil {
		elog.Error(1, fmt.Sprintf("%s service failed: %v", ServiceName, err))
		return
	}
	elog.Info(1, fmt.Sprintf("%s service stopped", ServiceName))
}

// InstallService installs the Windows service
func InstallService(exePath string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", ServiceName)
	}

	s, err = m.CreateService(ServiceName, exePath, mgr.Config{
		DisplayName: ServiceDisplayName,
		Description: ServiceDescription,
		StartType:   mgr.StartAutomatic, // Auto-start on boot
	})
	if err != nil {
		return err
	}
	defer s.Close()

	// Setup event log source
	err = eventlog.InstallAsEventCreate(ServiceName, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		// Non-fatal error
		fmt.Printf("Warning: could not install event log source: %v\n", err)
	}

	// Set recovery actions - restart on failure
	recoveryActions := []mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 10 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
	}
	err = s.SetRecoveryActions(recoveryActions, 86400) // Reset after 1 day
	if err != nil {
		fmt.Printf("Warning: failed to set recovery actions: %v\n", err)
	}

	return nil
}

// UninstallService removes the Windows service
func UninstallService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("service %s not installed", ServiceName)
	}
	defer s.Close()

	// Remove event log source
	eventlog.Remove(ServiceName)

	err = s.Delete()
	if err != nil {
		return err
	}
	return nil
}

// StartService starts the Windows service
func StartService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err != nil {
		return err
	}
	defer s.Close()

	return s.Start()
}

// StopService stops the Windows service
func StopService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err != nil {
		return err
	}
	defer s.Close()

	_, err = s.Control(svc.Stop)
	return err
}

// IsWindowsService checks if running as Windows service
func IsWindowsService() (bool, error) {
	return svc.IsWindowsService()
}
