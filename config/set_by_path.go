package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// SetByPath sets a config field at the given dotted path (e.g., "gmail.storage.driver")
// by walking the Config struct's YAML tags via reflection.
func SetByPath(cfg *Config, path, value string) error {
	segments := strings.Split(path, ".")
	if len(segments) == 0 {
		return fmt.Errorf("empty config path")
	}

	v := reflect.ValueOf(cfg).Elem()
	for i, seg := range segments {
		field, ok := findFieldByYAMLTag(v, seg)
		if !ok {
			return fmt.Errorf("unknown config key %q at segment %q", path, seg)
		}

		// If this is a pointer field, allocate if nil.
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			field = field.Elem()
		}

		// Last segment: set the value.
		if i == len(segments)-1 {
			return setField(field, value)
		}

		// Intermediate segment: descend into struct.
		if field.Kind() != reflect.Struct {
			return fmt.Errorf("cannot descend into non-struct field at %q", seg)
		}
		v = field
	}

	return nil
}

// findFieldByYAMLTag returns the struct field whose yaml tag matches seg.
func findFieldByYAMLTag(v reflect.Value, seg string) (reflect.Value, bool) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("yaml")
		name := strings.Split(tag, ",")[0]
		if name == seg {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

// setField sets a reflect.Value from a string, supporting string, bool, int, and int64.
func setField(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid bool value %q: %w", value, err)
		}
		field.SetBool(b)
	case reflect.Int, reflect.Int64:
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid int value %q: %w", value, err)
		}
		field.SetInt(n)
	case reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float value %q: %w", value, err)
		}
		field.SetFloat(f)
	default:
		return fmt.Errorf("unsupported field type %s", field.Kind())
	}
	return nil
}
