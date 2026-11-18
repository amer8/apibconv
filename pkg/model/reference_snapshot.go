package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
)

// fingerprint includes model extension fields, even when their normal JSON tags
// omit them. Those extensions are emitted by the format writers.
func fingerprint(value interface{}) string {
	if _, err := json.Marshal(value); err != nil {
		return "unavailable"
	}
	extensions := make(map[string]interface{})
	collectReferenceExtensions(reflect.ValueOf(value), "", extensions)
	data, err := json.Marshal(struct {
		Value      interface{}
		Extensions map[string]interface{}
	}{value, extensions})
	if err != nil {
		return "unavailable"
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func collectReferenceExtensions(value reflect.Value, path string, result map[string]interface{}) {
	if !value.IsValid() {
		return
	}
	switch value.Kind() {
	case reflect.Interface:
		// The only model-bearing interface field is Schema.AdditionalProperties.
		// Examples and other arbitrary values are opaque to this traversal.
		if !value.IsNil() && value.Elem().Kind() == reflect.Pointer && value.Elem().Type().Elem().PkgPath() == modelPackage {
			collectReferenceExtensions(value.Elem(), path, result)
		}
	case reflect.Pointer:
		if !value.IsNil() {
			collectReferenceExtensions(value.Elem(), path, result)
		}
	case reflect.Struct:
		if value.Type().PkgPath() != modelPackage {
			return
		}
		for i := 0; i < value.NumField(); i++ {
			field := value.Type().Field(i)
			if field.PkgPath != "" {
				continue
			}
			child := value.Field(i)
			childPath := path + "." + field.Name
			if child.Kind() == reflect.Interface && field.Name != "AdditionalProperties" {
				continue
			}
			if strings.HasSuffix(field.Name, "Extensions") && child.Kind() == reflect.Map {
				if !child.IsNil() {
					result[childPath] = child.Interface()
				}
				continue
			}
			collectReferenceExtensions(child, childPath, result)
		}
	case reflect.Map:
		if value.IsNil() || value.Type().Key().Kind() != reflect.String || value.Type().Elem().Kind() == reflect.Interface {
			return
		}
		iter := value.MapRange()
		for iter.Next() {
			collectReferenceExtensions(iter.Value(), path+"["+strconv.Quote(iter.Key().String())+"]", result)
		}
	case reflect.Slice, reflect.Array:
		if value.Type().Elem().Kind() == reflect.Interface {
			return
		}
		for i := 0; i < value.Len(); i++ {
			collectReferenceExtensions(value.Index(i), path+"["+strconv.Itoa(i)+"]", result)
		}
	}
}

// CaptureReference records the resolved fields at parse time so later edits are
// written inline instead of being silently hidden behind the original reference.
func (p *Parameter) CaptureReference() {
	if p.Ref != "" {
		p.ReferenceSnapshot = p.referenceFingerprint()
	}
}
func (p *Parameter) referenceFingerprint() string {
	value := *p
	value.Ref, value.ReferenceSnapshot = "", ""
	value.RefExtensions = nil
	return fingerprint(value)
}

// ReferenceUnchanged reports whether a parsed reference's resolved value was edited.
func (p *Parameter) ReferenceUnchanged() bool {
	return p.ReferenceSnapshot == "" || p.ReferenceSnapshot != "unavailable" && p.referenceFingerprint() == p.ReferenceSnapshot
}

// CaptureReference records the resolved fields of a referenced request body.
func (b *RequestBody) CaptureReference() {
	if b.Ref != "" {
		b.ReferenceSnapshot = b.referenceFingerprint()
	}
}
func (b *RequestBody) referenceFingerprint() string {
	value := *b
	value.Ref, value.ReferenceSnapshot = "", ""
	value.RefExtensions = nil
	return fingerprint(value)
}

// ReferenceUnchanged reports whether a parsed reference's resolved value was edited.
func (b *RequestBody) ReferenceUnchanged() bool {
	return b.ReferenceSnapshot == "" || b.ReferenceSnapshot != "unavailable" && b.referenceFingerprint() == b.ReferenceSnapshot
}

// CaptureReference records the resolved fields of a referenced response.
func (r *Response) CaptureReference() {
	if r.Ref != "" {
		r.ReferenceSnapshot = r.referenceFingerprint()
	}
}
func (r *Response) referenceFingerprint() string {
	value := *r
	value.Ref, value.ReferenceSnapshot = "", ""
	value.RefExtensions = nil
	return fingerprint(value)
}

// ReferenceUnchanged reports whether a parsed reference's resolved value was edited.
func (r *Response) ReferenceUnchanged() bool {
	return r.ReferenceSnapshot == "" || r.ReferenceSnapshot != "unavailable" && r.referenceFingerprint() == r.ReferenceSnapshot
}
