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
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/flag"
	"github.com/valentin-kaiser/go-core/logging"

	"gopkg.in/yaml.v2"
)

var (
	mutex      = &sync.RWMutex{}
	config     Config
	configname string
	onChange   []func(o Config, n Config) error
	lastChange atomic.Int64
	logger     = logging.GetPackageLogger("config")
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

func setEnvPrefix(prefix string) {
	mutex.Lock()
	defer mutex.Unlock()
	envPrefix = strings.ToUpper(prefix)
}

func setDefault(key string, value interface{}) {
	mutex.Lock()
	defer mutex.Unlock()
	lowerKey := strings.ToLower(key)
	defaults[lowerKey] = value
}

func bindPFlag(key string, flag *pflag.Flag) error {
	mutex.Lock()
	defer mutex.Unlock()
	flags[strings.ToLower(key)] = flag
	return nil
}

func setConfigName(name string) {
	mutex.Lock()
	defer mutex.Unlock()
	configname = name
}

func setConfigType(configTypeValue string) {
	mutex.Lock()
	defer mutex.Unlock()
	configType = configTypeValue
}

func addConfigPath(path string) {
	mutex.Lock()
	defer mutex.Unlock()
	configPath = path
}

func getEnvKey(key string) string {
	// Convert key from dot notation to env var format
	envKey := strings.ReplaceAll(key, ".", "_")
	envKey = strings.ReplaceAll(envKey, "-", "_")
	envKey = strings.ToUpper(envKey)

	if envPrefix == "" {
		return envKey
	}
	return envPrefix + "_" + envKey
}

func getValue(key string) interface{} {
	mutex.RLock()
	defer mutex.RUnlock()

	lowerKey := strings.ToLower(key)

	// Priority: flag > env var > config file > default

	// Check flag value first
	if flag, exists := flags[lowerKey]; exists && flag.Changed {
		return getFlagValue(flag)
	}

	// Check environment variable
	envKey := getEnvKey(key)
	if envVal := os.Getenv(envKey); envVal != "" {
		return envVal
	}

	// Check config file values
	if val, exists := values[lowerKey]; exists {
		return val
	}

	// Return default value
	if val, exists := defaults[lowerKey]; exists {
		return val
	}

	return nil
}

func getFlagValue(flag *pflag.Flag) interface{} {
	switch flag.Value.Type() {
	case "string":
		return flag.Value.String()
	case "int":
		val, err := strconv.Atoi(flag.Value.String())
		if err != nil {
			return flag.Value.String()
		}
		return val
	case "int8":
		val, err := strconv.ParseInt(flag.Value.String(), 10, 8)
		if err != nil {
			return flag.Value.String()
		}
		return int8(val)
	case "int16":
		val, err := strconv.ParseInt(flag.Value.String(), 10, 16)
		if err != nil {
			return flag.Value.String()
		}
		return int16(val)
	case "int32":
		val, err := strconv.ParseInt(flag.Value.String(), 10, 32)
		if err != nil {
			return flag.Value.String()
		}
		return int32(val)
	case "int64":
		val, err := strconv.ParseInt(flag.Value.String(), 10, 64)
		if err != nil {
			return flag.Value.String()
		}
		return val
	case "uint":
		val, err := strconv.ParseUint(flag.Value.String(), 10, 0)
		if err != nil {
			return flag.Value.String()
		}
		return uint(val)
	case "uint8":
		val, err := strconv.ParseUint(flag.Value.String(), 10, 8)
		if err != nil {
			return flag.Value.String()
		}
		return uint8(val)
	case "uint16":
		val, err := strconv.ParseUint(flag.Value.String(), 10, 16)
		if err != nil {
			return flag.Value.String()
		}
		return uint16(val)
	case "uint32":
		val, err := strconv.ParseUint(flag.Value.String(), 10, 32)
		if err != nil {
			return flag.Value.String()
		}
		return uint32(val)
	case "uint64":
		val, err := strconv.ParseUint(flag.Value.String(), 10, 64)
		if err != nil {
			return flag.Value.String()
		}
		return val
	case "float32":
		val, err := strconv.ParseFloat(flag.Value.String(), 32)
		if err != nil {
			return flag.Value.String()
		}
		return float32(val)
	case "float64":
		val, err := strconv.ParseFloat(flag.Value.String(), 64)
		if err != nil {
			return flag.Value.String()
		}
		return val
	case "bool":
		val, err := strconv.ParseBool(flag.Value.String())
		if err != nil {
			return flag.Value.String()
		}
		return val
	case "stringArray", "stringSlice":
		return strings.Split(flag.Value.String(), ",")
	}
	return flag.Value.String()
}

func readInConfig() error {
	mutex.Lock()
	defer mutex.Unlock()

	if configname == "" || configPath == "" {
		return apperror.NewError("config name and path must be set")
	}

	configFile := filepath.Join(configPath, configname+"."+configType)

	data, err := os.ReadFile(filepath.Clean(configFile))
	if err != nil {
		return err
	}

	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return err
	}

	// Flatten nested structure to dot notation
	values = make(map[string]interface{})
	flattenMap(yamlData, "")

	return nil
}

func flattenMap(data map[string]interface{}, prefix string) {
	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		if nested, ok := value.(map[string]interface{}); ok {
			flattenMap(nested, fullKey)
			continue
		}

		values[strings.ToLower(fullKey)] = value
	}
}

func unmarshalConfig(target interface{}) error {
	mutex.RLock()
	defer mutex.RUnlock()

	return unmarshalStruct(reflect.ValueOf(target), "")
}

func unmarshalStruct(v reflect.Value, prefix string) error {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		yamlTag := field.Tag.Get("yaml")
		if yamlTag == "-" {
			continue
		}

		fieldName := getFieldName(field)
		key := buildLabel(prefix, fieldName)

		if fieldValue.Kind() == reflect.Struct {
			if err := unmarshalStruct(fieldValue, key); err != nil {
				return err
			}
			continue
		}

		if fieldValue.Kind() == reflect.Ptr && fieldValue.Type().Elem().Kind() == reflect.Struct {
			if fieldValue.IsNil() {
				fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
			}
			if err := unmarshalStruct(fieldValue, key); err != nil {
				return err
			}
			continue
		}

		value := getValue(key)
		if value == nil {
			continue
		}

		if err := setFieldValue(fieldValue, value); err != nil {
			return err
		}
	}

	return nil
}

func setFieldValue(field reflect.Value, value interface{}) error {
	if !field.CanSet() {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		if str, ok := value.(string); ok {
			field.SetString(str)
			return nil
		}
		field.SetString(fmt.Sprintf("%v", value))

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if i, ok := value.(int); ok {
			field.SetInt(int64(i))
			return nil
		}
		if i, ok := value.(int64); ok {
			field.SetInt(i)
			return nil
		}
		if str, ok := value.(string); ok {
			if i, err := strconv.ParseInt(str, 10, 64); err == nil {
				field.SetInt(i)
			}
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if i, ok := value.(uint); ok {
			field.SetUint(uint64(i))
			return nil
		}
		if i, ok := value.(uint64); ok {
			field.SetUint(i)
			return nil
		}
		if str, ok := value.(string); ok {
			if i, err := strconv.ParseUint(str, 10, 64); err == nil {
				field.SetUint(i)
			}
		}

	case reflect.Float32, reflect.Float64:
		if f, ok := value.(float64); ok {
			field.SetFloat(f)
			return nil
		}
		if f, ok := value.(float32); ok {
			field.SetFloat(float64(f))
			return nil
		}
		if str, ok := value.(string); ok {
			if f, err := strconv.ParseFloat(str, 64); err == nil {
				field.SetFloat(f)
			}
		}

	case reflect.Bool:
		if b, ok := value.(bool); ok {
			field.SetBool(b)
			return nil
		}
		if str, ok := value.(string); ok {
			if b, err := strconv.ParseBool(str); err == nil {
				field.SetBool(b)
			}
		}

	case reflect.Slice:
		if field.Type().Elem().Kind() != reflect.String {
			return nil
		}
		if slice, ok := value.([]string); ok {
			field.Set(reflect.ValueOf(slice))
			return nil
		}
		if slice, ok := value.([]interface{}); ok {
			strSlice := make([]string, len(slice))
			for i, v := range slice {
				strSlice[i] = fmt.Sprintf("%v", v)
			}
			field.Set(reflect.ValueOf(strSlice))
		}
	}

	return nil
}

func watchConfig(onChange func(fsnotify.Event)) error {
	mutex.Lock()
	defer mutex.Unlock()

	if watcher != nil {
		watcher.Close()
	}

	newWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	watcher = newWatcher

	configFile := filepath.Join(configPath, configname+"."+configType)

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

func resetConfig() {
	mutex.Lock()
	defer mutex.Unlock()

	if watcher != nil {
		watcher.Close()
		watcher = nil
	}

	envPrefix = ""
	defaults = make(map[string]interface{})
	values = make(map[string]interface{})
	flags = make(map[string]*pflag.Flag)
	configPath = ""
	configType = ""
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
	// SetTypeByDefaultValue and AutomaticEnv are handled automatically by our implementation

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

// set applies the configuration to the global variable
func set(appConfig Config) {
	mutex.Lock()
	defer mutex.Unlock()
	config = appConfig
}

// save saves the configuration to the file
// If the file does not exist, it creates a new one with the default values
func save() error {
	// Ensure the directory exists before trying to create the file
	if err := os.MkdirAll(flag.Path, 0750); err != nil {
		return apperror.NewError("creating configuration directory failed").AddError(err)
	}

	path, err := filepath.Abs(filepath.Join(flag.Path, configname+".yaml"))
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

// kebabCase converts a string to kebab-case (dash-separated lowercase)
// Example: "ApplicationName" -> "application-name"
func kebabCase(s string) string {
	if s == "" {
		return s
	}
	
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteByte('-')
			}
			result.WriteRune(unicode.ToLower(r))
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

// declareFlag declares a flag with the given label, usage and default value
// It also binds the flag to the configuration
func declareFlag(label string, usage string, defaultValue interface{}) error {
	setDefault(label, defaultValue)
	pflagLabel := kebabCase(label)
	label = strings.ToLower(label)

	// Check if flag already exists to avoid redefinition errors
	if pflag.Lookup(pflagLabel) != nil {
		// Flag already exists, just bind to config
		return bindPFlag(label, pflag.Lookup(pflagLabel))
	}

	switch v := defaultValue.(type) {
	case string:
		pflag.String(pflagLabel, v, usage)
	case int:
		pflag.Int(pflagLabel, v, usage)
	case uint:
		pflag.Uint(pflagLabel, v, usage)
	case int8:
		pflag.Int8(pflagLabel, v, usage)
	case uint8:
		pflag.Uint8(pflagLabel, v, usage)
	case int16:
		pflag.Int16(pflagLabel, v, usage)
	case uint16:
		pflag.Uint16(pflagLabel, v, usage)
	case int32:
		pflag.Int32(pflagLabel, v, usage)
	case uint32:
		pflag.Uint32(pflagLabel, v, usage)
	case int64:
		pflag.Int64(pflagLabel, v, usage)
	case uint64:
		pflag.Uint64(pflagLabel, v, usage)
	case float32:
		pflag.Float32(pflagLabel, v, usage)
	case float64:
		pflag.Float64(pflagLabel, v, usage)
	case bool:
		pflag.Bool(pflagLabel, v, usage)
	case []string:
		pflag.StringArray(pflagLabel, v, usage)
	default:
		return nil
	}

	return bindPFlag(label, pflag.Lookup(pflagLabel))
}

// parseStructTags parses the struct tags of the given struct and registers the flags
// It also sets the default values of the flags to the values of the struct fields
func parseStructTags(v reflect.Value, labelBase string) error {
	// If the config is a pointer, we need to get the type of the element
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		// If the field is not exported, we skip it
		if t.Field(i).PkgPath != "" {
			continue
		}

		field := t.Field(i)
		fieldName := getFieldName(field)

		// If the field is a pointer, we need to dereference it
		if v.Field(i).Kind() == reflect.Ptr {
			if v.Field(i).IsNil() {
				v.Field(i).Set(reflect.New(field.Type.Elem()))
			}

			if err := parseStructTags(v.Field(i).Elem(), fieldName); err != nil {
				return apperror.Wrap(err)
			}
			continue
		}

		// If the field is a struct, we need to iterate over its fields
		if field.Type.Kind() == reflect.Struct {
			subv := v.Field(i)
			if subv.Kind() == reflect.Ptr {
				subv = subv.Elem()
			}

			label := buildLabel(labelBase, fieldName)
			if err := parseStructTags(subv, label); err != nil {
				return apperror.Wrap(err)
			}
			continue
		}

		tag := buildLabel(labelBase, fieldName)
		if err := declareFlag(tag, field.Tag.Get("usage"), v.Field(i).Interface()); err != nil {
			return apperror.Wrap(err)
		}
	}

	return nil
}

// getFieldName extracts the field name from struct field, preferring yaml tag over field name
func getFieldName(field reflect.StructField) string {
	yamlTag := field.Tag.Get("yaml")
	if yamlTag == "" || yamlTag == "-" {
		return field.Name
	}

	if idx := strings.Index(yamlTag, ","); idx != -1 {
		return yamlTag[:idx]
	}
	return yamlTag
}

// buildLabel constructs a dot-separated label from base and field name
func buildLabel(base, fieldName string) string {
	if base == "" {
		return fieldName
	}
	return base + "." + fieldName
}

// Reset clears all global state - primarily for testing purposes
func Reset() {
	mutex.Lock()
	defer mutex.Unlock()
	config = nil
	configname = ""
	onChange = nil
	lastChange.Store(0)

	// Reset config manager state (already holding lock)
	if watcher != nil {
		watcher.Close()
		watcher = nil
	}

	envPrefix = ""
	defaults = make(map[string]interface{})
	values = make(map[string]interface{})
	flags = make(map[string]*pflag.Flag)
	configPath = ""
	configType = ""
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
