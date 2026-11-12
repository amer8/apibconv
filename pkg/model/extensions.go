package model

import (
	"reflect"
	"strings"
)

// WithoutExtensions returns a copy of the model without vendor extensions.
// Model objects are copied recursively; arbitrary example and extension values
// are not traversed. Cycles between model objects are preserved.
func (a *API) WithoutExtensions() *API {
	if a == nil {
		return nil
	}
	return cloneWithoutExtensions(reflect.ValueOf(a), make(map[interface{}]reflect.Value)).Interface().(*API)
}

var modelPackage = reflect.TypeOf(API{}).PkgPath()

func cloneWithoutExtensions(value reflect.Value, seen map[interface{}]reflect.Value) reflect.Value {
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return value
		}
		inner := value.Elem()
		if inner.Kind() != reflect.Pointer || inner.Type().Elem().PkgPath() != modelPackage {
			return value
		}
		result := reflect.New(value.Type()).Elem()
		result.Set(cloneWithoutExtensions(inner, seen))
		return result
	case reflect.Pointer:
		if value.IsNil() || value.Type().Elem().PkgPath() != modelPackage {
			return value
		}
		key := value.Interface()
		if existing, ok := seen[key]; ok {
			return existing
		}
		result := reflect.New(value.Type().Elem())
		seen[key] = result
		result.Elem().Set(cloneWithoutExtensions(value.Elem(), seen))
		return result
	case reflect.Struct:
		if value.Type().PkgPath() != modelPackage {
			return value
		}
		result := reflect.New(value.Type()).Elem()
		for i := 0; i < value.NumField(); i++ {
			name := value.Type().Field(i).Name
			if strings.HasSuffix(name, "Extensions") {
				continue
			}
			if value.Field(i).Kind() == reflect.Interface && name != "AdditionalProperties" {
				result.Field(i).Set(value.Field(i))
				continue
			}
			result.Field(i).Set(cloneWithoutExtensions(value.Field(i), seen))
		}
		return result
	case reflect.Slice:
		if value.IsNil() {
			return value
		}
		result := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := 0; i < value.Len(); i++ {
			result.Index(i).Set(cloneWithoutExtensions(value.Index(i), seen))
		}
		return result
	case reflect.Map:
		if value.IsNil() || value.Type().Elem().Kind() == reflect.Interface {
			return value
		}
		result := reflect.MakeMapWithSize(value.Type(), value.Len())
		iter := value.MapRange()
		for iter.Next() {
			result.SetMapIndex(iter.Key(), cloneWithoutExtensions(iter.Value(), seen))
		}
		return result
	default:
		return value
	}
}
