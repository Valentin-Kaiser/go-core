package logging

import (
	"context"
	"sync"
)

var (
	// global is the default adapter used when no package-specific adapter is set
	global Adapter = NewNoOpAdapter()
	// packages stores package-specific adapters
	packages sync.Map
	// mu protects the global adapter
	mu sync.RWMutex
)

// SetGlobalAdapter sets the global logging adapter for all packages
// This will be used as the default for all packages unless they have a specific adapter
func SetGlobalAdapter(adapter Adapter) {
	mu.Lock()
	defer mu.Unlock()
	global = adapter
}

// GetGlobalAdapter returns the current global adapter
func GetGlobalAdapter() Adapter {
	mu.RLock()
	defer mu.RUnlock()
	return global
}

// SetPackageAdapter sets a specific adapter for a package
// This overrides the global adapter for the specified package
func SetPackageAdapter(pkg string, adapter Adapter) {
	packages.Store(pkg, adapter)
}

// DynamicAdapter wraps the package lookup to always use the current adapter
type DynamicAdapter struct {
	pkg string
}

// NewDynamicAdapter creates a dynamic adapter for a package
func NewDynamicAdapter(pkg string) Adapter {
	return &DynamicAdapter{pkg: pkg}
}

func (d *DynamicAdapter) SetLevel(level Level) {
	d.current().SetLevel(level)
}

func (d *DynamicAdapter) GetLevel() Level {
	return d.current().GetLevel()
}

func (d *DynamicAdapter) Trace(msg string, fields ...Field) {
	d.current().Trace(msg, fields...)
}

func (d *DynamicAdapter) Debug(msg string, fields ...Field) {
	d.current().Debug(msg, fields...)
}

func (d *DynamicAdapter) Info(msg string, fields ...Field) {
	d.current().Info(msg, fields...)
}

func (d *DynamicAdapter) Warn(msg string, fields ...Field) {
	d.current().Warn(msg, fields...)
}

func (d *DynamicAdapter) Error(msg string, fields ...Field) {
	d.current().Error(msg, fields...)
}

func (d *DynamicAdapter) Fatal(msg string, fields ...Field) {
	d.current().Fatal(msg, fields...)
}

func (d *DynamicAdapter) Panic(msg string, fields ...Field) {
	d.current().Panic(msg, fields...)
}

func (d *DynamicAdapter) Printf(format string, v ...interface{}) {
	d.current().Printf(format, v...)
}

func (d *DynamicAdapter) WithContext(ctx context.Context) Adapter {
	return d.current().WithContext(ctx)
}

func (d *DynamicAdapter) WithFields(fields ...Field) Adapter {
	return d.current().WithFields(fields...)
}

func (d *DynamicAdapter) WithPackage(pkg string) Adapter {
	return d.current().WithPackage(pkg)
}

// GetPackageLogger returns a logger for a specific package
// Returns a dynamic adapter that will always use the current global/package-specific adapter
func GetPackageLogger(pkg string) Adapter {
	return NewDynamicAdapter(pkg)
}

// DisablePackage disables logging for a specific package
func DisablePackage(pkg string) {
	SetPackageAdapter(pkg, NewNoOpAdapter())
}

// EnablePackage removes package-specific adapter, falling back to global
func EnablePackage(pkg string) {
	packages.Delete(pkg)
}

// SetPackageLevel sets the log level for a specific package
// If the package doesn't have a specific adapter, this creates one based on the global adapter
func SetPackageLevel(pkg string, level Level) {
	if adapter, ok := packages.Load(pkg); ok {
		adapter.(Adapter).SetLevel(level)
		return
	}

	// Create a package-specific adapter based on the global one
	mu.RLock()
	var newAdapter Adapter
	switch adapter := global.(type) {
	case *ZerologAdapter:
		newAdapter = NewZerologAdapterWithLogger(adapter.logger)
	default:
		newAdapter = NewNoOpAdapter() // Fallback to NoOpAdapter if unknown type
	}
	mu.RUnlock()

	newAdapter.SetLevel(level)
	SetPackageAdapter(pkg, newAdapter.WithPackage(pkg))
}

// GetPackageLevel returns the log level for a specific package
func GetPackageLevel(pkg string) Level {
	return GetPackageLogger(pkg).GetLevel()
}

// ListPackages returns all packages that have specific adapters
func ListPackages() []string {
	var p []string
	packages.Range(func(key, value interface{}) bool {
		if pkg, ok := key.(string); ok {
			p = append(p, pkg)
		}
		return true
	})
	return p
}

// current returns the current adapter for this package
func (d *DynamicAdapter) current() Adapter {
	if adapter, ok := packages.Load(d.pkg); ok {
		return adapter.(Adapter)
	}

	mu.RLock()
	defer mu.RUnlock()
	return global.WithPackage(d.pkg)
}
