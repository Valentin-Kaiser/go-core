// Package config provides a simple, structured, and extensible way to manage
// application configuration in Go.
//
// It builds upon the Viper library and adds
// powerful features like validation, dynamic watching, default value registration,
// environment and flag integration, and structured config registration.
//
// Key Features:
//
//   - Register typed configuration structs with default values.
//   - Parse YAML configuration files and bind fields to CLI flags and environment variables.
//   - Automatically generate flags based on struct field tags.
//   - Validate configuration using custom logic (via `Validate()` method).
//   - Watch configuration files for changes and hot-reload updated values.
//   - Write current configuration back to disk.
//   - Automatically fallbacks to default config creation if no file is found.
//
// All configuration structs must implement the `Config` interface:
//
//	type Config interface {
//	    Validate() error
//	}
//
// Example:
//
//	package config
//
//	import (
//	    "fmt"
//	    "github.com/valentin-kaiser/go-core/config"
//	    "github.com/fsnotify/fsnotify"
//	)
//
//	type ServerConfig struct {
//	    Host string `yaml:"host" usage:"The host of the server"`
//	    Port int    `yaml:"port" usage:"The port of the server"`
//	}
//
//	func (c *ServerConfig) Validate() error {
//	    if c.Host == "" {
//	        return fmt.Errorf("host cannot be empty")
//	    }
//	    if c.Port <= 0 {
//	        return fmt.Errorf("port must be greater than 0")
//	    }
//	    return nil
//	}
//
//	func Get() *ServerConfig {
//	    c, ok := config.Get().(*ServerConfig)
//	    if !ok {
//	        return &ServerConfig{}
//	    }
//	    return c
//	}
//
//	func init() {
//	    cfg := &ServerConfig{
//	        Host: "localhost",
//	        Port: 8080,
//	    }
//
//	    if err := config.Register("server", cfg); err != nil {
//	        fmt.Println("Error registering config:", err)
//	        return
//	    }
//
//	    if err := config.Read(); err != nil {
//	        fmt.Println("Error reading config:", err)
//	        return
//	    }
//
//	    config.Watch(func(e fsnotify.Event) {
//	        if err := config.Read(); err != nil {
//	            fmt.Println("Error reloading config:", err)
//	        }
//	    })
//
//	    if err := config.Write(cfg); err != nil {
//	        fmt.Println("Error writing config:", err)
//	    }
//	}
package config

import (
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/flag"
	"github.com/valentin-kaiser/go-core/logging"
)

var (
	logger     = logging.GetPackageLogger("config")
	mutex      = &sync.RWMutex{}
	config     Config
	configname string
	onChange   []func(o Config, n Config) error
	lastChange atomic.Int64
	envPrefix  string
	defaults   map[string]interface{}
	values     map[string]interface{}
	flags      map[string]*pflag.Flag
	watcher    *fsnotify.Watcher
	configPath string
	configType string
)

func init() {
	defaults = make(map[string]interface{})
	values = make(map[string]interface{})
	flags = make(map[string]*pflag.Flag)
}

// Config is the interface that all configuration structs must implement
// It should contain a Validate method that checks the configuration for errors
type Config interface {
	Validate() error
}

// Register registers a configuration struct and parses its tags
// The name is used as the name of the configuration file and the prefix for the environment variables
func Register(name string, c Config) error {
	if c == nil {
		return apperror.NewError("the configuration provided is nil")
	}

	if reflect.TypeOf(c).Kind() != reflect.Ptr || reflect.TypeOf(c).Elem().Kind() != reflect.Struct {
		return apperror.NewErrorf("the configuration provided is not a pointer to a struct, got %T", c)
	}

	configname = name
	setEnvPrefix(strings.ReplaceAll(configname, "-", "_"))

	err := parseStructTags(reflect.ValueOf(c), "")
	if err != nil {
		return apperror.Wrap(err)
	}

	set(c)
	return nil
}

// OnChange registers a function that is called when the configuration changes
func OnChange(f func(o Config, n Config) error) {
	onChange = append(onChange, f)
}

// Get returns the current configuration
func Get() Config {
	mutex.RLock()
	defer mutex.RUnlock()
	return config
}

// Read reads the configuration from the file, validates it and applies it
// If the file does not exist, it creates a new one with the default values
func Read() error {
	setConfigName(configname)
	setConfigType("yaml")
	addConfigPath(flag.Path)

	err := readInConfig()
	if err != nil {
		err := os.MkdirAll(flag.Path, 0750)
		if err != nil {
			return apperror.NewError("creating configuration directory failed").AddError(err)
		}

		err = save()
		if err != nil {
			return apperror.NewError("writing default configuration file failed").AddError(err)
		}

		// Retry reading the config file after creating it
		err = readInConfig()
		if err != nil {
			return apperror.NewError("reading configuration file after creation failed").AddError(err)
		}
	}

	change, ok := reflect.New(reflect.TypeOf(config).Elem()).Interface().(Config)
	if !ok {
		return apperror.NewErrorf("creating new instance of %T failed", config)
	}

	err = unmarshalConfig(change)
	if err != nil {
		return apperror.NewErrorf("unmarshalling configuration data in %T failed", config).AddError(err)
	}

	err = change.Validate()
	if err != nil {
		return apperror.Wrap(err)
	}

	o := Get()
	set(change)
	for _, f := range onChange {
		err = f(o, change)
		if err != nil {
			return apperror.Wrap(err)
		}
	}

	return nil
}

// Write writes the configuration to the file, validates it and applies it
// If the file does not exist, it creates a new one with the default values
func Write(change Config) error {
	if change == nil {
		return apperror.NewError("the configuration provided is nil")
	}

	err := change.Validate()
	if err != nil {
		return apperror.Wrap(err)
	}

	o := Get()
	set(change)
	err = save()
	if err != nil {
		return apperror.Wrap(err)
	}

	for _, f := range onChange {
		err = f(o, change)
		if err != nil {
			return apperror.Wrap(err)
		}
	}

	return nil
}

// Watch watches the configuration file for changes and calls the provided function when it changes
// It ignores changes that happen within 1 second of each other
// This is to prevent multiple calls when the file is saved
func Watch() {
	err := watchConfig(func(_ fsnotify.Event) {
		if time.Now().UnixMilli()-lastChange.Load() < 1000 {
			return
		}
		lastChange.Store(time.Now().UnixMilli())
		err := Read()
		if err != nil {
			logger.Error().Err(err).Msg("failed to read configuration")
			return
		}
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to setup config watcher")
	}
}

// Reset clears the global state of the config package
func Reset() {
	mutex.Lock()
	defer mutex.Unlock()
	config = nil
	configname = ""
	onChange = nil
	lastChange.Store(0)

	resetConfig()
}

// Changed checks if two configuration values are different by comparing their reflection values.
// It returns true if the configurations differ, false if they are the same.
// This function handles nil values correctly and performs deep comparison of the underlying values.
func Changed(o, n any) bool {
	if o == nil && n == nil {
		return false
	}

	if o == nil || n == nil {
		return true
	}

	ov := reflect.ValueOf(o)
	nv := reflect.ValueOf(n)

	if ov.Kind() == reflect.Ptr && !ov.IsNil() {
		ov = ov.Elem()
	}
	if nv.Kind() == reflect.Ptr && !nv.IsNil() {
		nv = nv.Elem()
	}

	if ov.Kind() != nv.Kind() {
		return true
	}

	return !reflect.DeepEqual(ov.Interface(), nv.Interface())
}

// set applies the configuration to the global variable
func set(appConfig Config) {
	mutex.Lock()
	defer mutex.Unlock()
	config = appConfig
}
