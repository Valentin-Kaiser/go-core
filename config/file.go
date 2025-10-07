package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/flag"

	"gopkg.in/yaml.v2"
)

func setConfigName(name string) {
	mutex.Lock()
	defer mutex.Unlock()
	configName = name
}

func addConfigPath(path string) {
	mutex.Lock()
	defer mutex.Unlock()
	configPath = path
}

func read() error {
	mutex.Lock()
	defer mutex.Unlock()

	if configName == "" || configPath == "" {
		return apperror.NewError("config name and path must be set")
	}

	configFile := filepath.Join(configPath, configName+".yaml")

	data, err := os.ReadFile(filepath.Clean(configFile))
	if err != nil {
		return err
	}

	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return err
	}

	values = make(map[string]interface{})
	flatten(yamlData, "")

	return nil
}

func watch(onChange func(fsnotify.Event)) error {
	mutex.Lock()
	defer mutex.Unlock()

	if watcher != nil {
		watcher.Close()
	}

	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	configFile := filepath.Join(configPath, configName+".yaml")
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Name == configFile && (event.Op&fsnotify.Write == fsnotify.Write) {
					onChange(event)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Error().Err(err).Msg("config file watcher error")
			}
		}
	}()

	return watcher.Add(configPath)
}

// save saves the configuration to the file
// If the file does not exist, it creates a new one with the default values
func save() error {
	// Ensure the directory exists before trying to create the file
	if err := os.MkdirAll(flag.Path, 0750); err != nil {
		return apperror.NewError("creating configuration directory failed").AddError(err)
	}

	path, err := filepath.Abs(filepath.Join(flag.Path, configName+".yaml"))
	if err != nil {
		return apperror.NewError("building absolute path of configuration file failed").AddError(err)
	}
	file, err := os.OpenFile(filepath.Clean(path), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return apperror.NewError("opening configuration file failed").AddError(err)
	}

	mutex.RLock()
	defer mutex.RUnlock()
	data, err := yaml.Marshal(config)
	if err != nil {
		return apperror.NewError("marshalling configuration data failed").AddError(err)
	}

	_, err = file.Write(data)
	if err != nil {
		return apperror.NewError("writing configuration data to file failed").AddError(err)
	}

	err = file.Close()
	if err != nil {
		return apperror.NewError("closing configuration file failed").AddError(err)
	}

	return nil
}

func flatten(data map[string]interface{}, prefix string) {
	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		if nested, ok := value.(map[string]interface{}); ok {
			flatten(nested, fullKey)
			continue
		}

		values[strings.ToLower(fullKey)] = value
	}
}
