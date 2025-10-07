package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func getValue(key string) interface{} {
	mutex.RLock()
	defer mutex.RUnlock()

	lowerKey := strings.ToLower(key)
	if flag, exists := flags[lowerKey]; exists && flag.Changed {
		return getFlagValue(flag)
	}

	envKey := getFlagKey(key)
	if envVal := os.Getenv(envKey); envVal != "" {
		return envVal
	}

	if val, exists := values[lowerKey]; exists {
		return val
	}

	if val, exists := defaults[lowerKey]; exists {
		return val
	}

	return nil
}

func unmarshal(target interface{}) error {
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
