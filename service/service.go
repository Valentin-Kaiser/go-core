// Package service provides a simple way to create and manage system services (daemons) in Go applications.
package service

import (
	"os"
	"path/filepath"

	"github.com/Valentin-Kaiser/go-core/apperror"
	"github.com/Valentin-Kaiser/go-core/flag"
	"github.com/Valentin-Kaiser/go-core/logging"
	"github.com/kardianos/service"
)

var (
	interactive = false
	logger      = logging.GetPackageLogger("service")
)

// Config is an alias for service.Config providing system service configuration options.
// It includes service name, display name, description, dependencies, and platform-specific settings.
type Config = service.Config

func init() {
	var err error
	interactive = service.AvailableSystems()[len(service.AvailableSystems())-1].Interactive()
	if !interactive {
		flag.Path, err = os.Executable()
		if err != nil {
			logger.Error().Fields(logging.F("error", err)).Msg("creating service failed")
			return
		}
		flag.Path = filepath.Join(filepath.Dir(flag.Path), "data")
	}

	flag.Path, err = filepath.Abs(flag.Path)
	if err != nil {
		logger.Error().Fields(logging.F("error", err)).Msg("determining absolute path failed")
		return
	}
}

// Run starts a service with the provided configuration and start/stop functions.
// It handles signal management and graceful shutdown.
func Run(config *Config, start func(s *Service) error, stop func(s *Service) error) error {
	s := &Service{
		err:   make(chan error, 1),
		start: start,
		stop:  stop,
	}

	if config == nil {
		return apperror.NewError("service config is nil")
	}

	if start == nil {
		return apperror.NewError("service start function is nil")
	}

	if stop == nil {
		return apperror.NewError("service stop function is nil")
	}

	if interactive {
		err := s.Start(nil)
		if err != nil {
			return err
		}

		return apperror.Wrap(<-s.err)
	}

	svc, err := service.New(s, config)
	if err != nil {
		return apperror.NewError("creating service failed").AddError(err)
	}

	err = svc.Run()
	if err != nil {
		return apperror.NewError("starting service failed").AddError(err)
	}

	return apperror.Wrap(<-s.err)
}

// IsInteractive returns true if the service is running in interactive mode
func IsInteractive() bool {
	return interactive
}

// Service wraps the kardianos/service.Service interface and provides additional
// functionality for managing application lifecycle with custom start and stop handlers.
// It includes error handling and graceful shutdown capabilities.
type Service struct {
	service.Service
	err   chan error
	start func(s *Service) error
	stop  func(s *Service) error
}

// Start implements the service.Interface Start method and executes the custom start handler.
// It is called by the service manager when the service should start.
func (s *Service) Start(svc service.Service) error {
	s.Service = svc
	if s.start == nil {
		return apperror.NewError("service start function is not defined")
	}
	go func() {
		s.err <- s.start(s)
	}()
	return nil
}

// Stop implements the service.Interface Stop method and executes the custom stop handler.
// It is called by the service manager when the service should stop gracefully.
func (s *Service) Stop(svc service.Service) error {
	s.Service = svc
	if s.stop == nil {
		return apperror.NewError("service stop function is not defined")
	}

	go func() {
		s.err <- s.stop(s)
	}()
	return nil
}
