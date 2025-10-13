// Package ptr provides helper functions to get pointers to basic types.
package ptr

func Float32Ptr(f float32) *float32 { return &f }
func Float64Ptr(f float64) *float64 { return &f }
func IntPtr(i int) *int             { return &i }
func Int8Ptr(i int8) *int8          { return &i }
func Int16Ptr(i int16) *int16       { return &i }
func Int32Ptr(i int32) *int32       { return &i }
func Int64Ptr(i int64) *int64       { return &i }
func UintPtr(i uint) *uint          { return &i }
func Uint8Ptr(i uint8) *uint8       { return &i }
func Uint16Ptr(i uint16) *uint16    { return &i }
func Uint32Ptr(i uint32) *uint32    { return &i }
func Uint64Ptr(i uint64) *uint64    { return &i }
func StringPtr(s string) *string    { return &s }
func BoolPtr(b bool) *bool          { return &b }
