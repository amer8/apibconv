package openapi

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

// referenceName resolves a single JSON Pointer token in a local component map.
func referenceName(ref, prefix string) (string, error) {
	if !strings.HasPrefix(ref, prefix) {
		return "", fmt.Errorf("unsupported reference %.128q", ref)
	}
	token := strings.TrimPrefix(ref, prefix)
	if strings.Contains(token, "/") {
		return "", fmt.Errorf("unsupported reference %.128q", ref)
	}
	decoded, err := url.PathUnescape(token)
	if err != nil {
		return "", fmt.Errorf("invalid reference %.128q: %w", ref, err)
	}
	for i := 0; i < len(decoded); i++ {
		if decoded[i] == '~' {
			if i+1 == len(decoded) || (decoded[i+1] != '0' && decoded[i+1] != '1') {
				return "", fmt.Errorf("invalid JSON pointer %.128q", ref)
			}
			i++
		}
	}
	return strings.ReplaceAll(strings.ReplaceAll(decoded, "~1", "/"), "~0", "~"), nil
}

// Extensions on the referring object take precedence over the referenced object.
func mergeReferenceExtensions(base map[string]interface{}, overlays []map[string]interface{}) map[string]interface{} {
	result := clonedExtensions(base)
	for i := len(overlays) - 1; i >= 0; i-- {
		for key, value := range overlays[i] {
			if strings.HasPrefix(key, "x-") {
				if result == nil {
					result = make(map[string]interface{})
				}
				result[key] = value
			}
		}
	}
	return result
}

func resolveParameter(input *Parameter, values map[string]Parameter, prefix string, budget *callbackResolver) (Parameter, error) {
	v := *input
	sourceRef, sourceExtensions := v.Ref, v.Extensions
	seen := map[string]bool{}
	var overlays []map[string]interface{}
	for v.Ref != "" {
		if err := budget.reserveResolution(v.Ref, v.Extensions); err != nil {
			return Parameter{}, err
		}
		overlays = append(overlays, v.Extensions)
		if seen[v.Ref] {
			return Parameter{}, fmt.Errorf("cyclic parameter reference %.128q", v.Ref)
		}
		seen[v.Ref] = true
		name, err := referenceName(v.Ref, prefix)
		if err != nil {
			return Parameter{}, err
		}
		target, ok := values[name]
		if !ok {
			return Parameter{}, fmt.Errorf("parameter reference %.128q not found", v.Ref)
		}
		v = target
	}
	if err := budget.reserveResolution(v.Ref, v.Extensions); err != nil {
		return Parameter{}, err
	}
	v.Extensions = mergeReferenceExtensions(v.Extensions, overlays)
	if sourceRef != "" {
		v.sourceRef, v.sourceRefExtensions = sourceRef, clonedExtensions(sourceExtensions)
	}
	return v, nil
}

//nolint:gocritic // Responses are copied deliberately when converting or resolving them.
func resolveResponse(v Response, values map[string]Response, prefix string, budget *callbackResolver) (Response, error) {
	sourceRef, sourceExtensions := v.Ref, v.Extensions
	seen := map[string]bool{}
	var overlays []map[string]interface{}
	for v.Ref != "" {
		if err := budget.reserveResolution(v.Ref, v.Extensions); err != nil {
			return Response{}, err
		}
		overlays = append(overlays, v.Extensions)
		if seen[v.Ref] {
			return Response{}, fmt.Errorf("cyclic response reference %.128q", v.Ref)
		}
		seen[v.Ref] = true
		name, err := referenceName(v.Ref, prefix)
		if err != nil {
			return Response{}, err
		}
		target, ok := values[name]
		if !ok {
			return Response{}, fmt.Errorf("response reference %.128q not found", v.Ref)
		}
		v = target
	}
	if err := budget.reserveResolution(v.Ref, v.Extensions); err != nil {
		return Response{}, err
	}
	v.Extensions = mergeReferenceExtensions(v.Extensions, overlays)
	if sourceRef != "" {
		v.sourceRef, v.sourceRefExtensions = sourceRef, clonedExtensions(sourceExtensions)
	}
	return v, nil
}

//nolint:gocritic // A value copy preserves the existing conversion and marshaling behavior.
func resolveRequestBody(v RequestBody, values map[string]RequestBody, budget *callbackResolver) (RequestBody, error) {
	sourceRef, sourceExtensions := v.Ref, v.Extensions
	seen := map[string]bool{}
	var overlays []map[string]interface{}
	for v.Ref != "" {
		if err := budget.reserveResolution(v.Ref, v.Extensions); err != nil {
			return RequestBody{}, err
		}
		overlays = append(overlays, v.Extensions)
		if seen[v.Ref] {
			return RequestBody{}, fmt.Errorf("cyclic request body reference %.128q", v.Ref)
		}
		seen[v.Ref] = true
		name, err := referenceName(v.Ref, "#/components/requestBodies/")
		if err != nil {
			return RequestBody{}, err
		}
		target, ok := values[name]
		if !ok {
			return RequestBody{}, fmt.Errorf("request body reference %.128q not found", v.Ref)
		}
		v = target
	}
	if err := budget.reserveResolution(v.Ref, v.Extensions); err != nil {
		return RequestBody{}, err
	}
	v.Extensions = mergeReferenceExtensions(v.Extensions, overlays)
	if sourceRef != "" {
		v.sourceRef, v.sourceRefExtensions = sourceRef, clonedExtensions(sourceExtensions)
	}
	return v, nil
}

func resolveOperation(op *Operation, params map[string]Parameter, responses map[string]Response, bodies map[string]RequestBody, v2 bool, budget *callbackResolver) error {
	if op == nil {
		return nil
	}
	paramPrefix, respPrefix := "#/components/parameters/", "#/components/responses/"
	if v2 {
		paramPrefix, respPrefix = "#/parameters/", "#/responses/"
	}
	for i := range op.Parameters {
		p, err := resolveParameter(&op.Parameters[i], params, paramPrefix, budget)
		if err != nil {
			return err
		}
		op.Parameters[i] = p
	}
	if op.RequestBody != nil {
		b, err := resolveRequestBody(*op.RequestBody, bodies, budget)
		if err != nil {
			return err
		}
		op.RequestBody = &b
	}
	for k, v := range op.Responses {
		r, err := resolveResponse(v, responses, respPrefix, budget)
		if err != nil {
			return err
		}
		op.Responses[k] = r
	}
	return nil
}

// Each visit can materialize another copy of the referenced callback in the
// model. Share a budget across components, webhooks, paths and sibling branches.
const (
	maxCallbackVisits     = 4096
	maxCallbackModelBytes = 32 << 20
)

type callbackResolver struct {
	doc        *OpenAPI
	remaining  int
	modelBytes int64
}

// Outside callbacks, component resolution has no callback expansion budget.
// Inside, reserve room for each shallow map copy and reference-chain entry.
func (r *callbackResolver) reserveResolution(ref string, ext map[string]interface{}) error {
	if r != nil {
		r.modelBytes -= 64 + int64(len(ref)) + int64(len(ext))*128
		for key := range ext {
			r.modelBytes -= int64(len(key))
		}
		if r.modelBytes < 0 {
			return fmt.Errorf("callback model expansion limit exceeded")
		}
	}
	return nil
}

func (r *callbackResolver) charge() error {
	if r.remaining == 0 {
		return fmt.Errorf("callback expansion limit exceeded")
	}
	r.remaining--
	return nil
}

func (r *callbackResolver) resolvePath(input *PathItem, depth int) (PathItem, error) {
	doc := r.doc
	var budget *callbackResolver
	if depth > 0 {
		budget = r
		if err := r.charge(); err != nil {
			return PathItem{}, err
		}
	}
	if depth > 64 {
		return PathItem{}, fmt.Errorf("callback nesting or reference cycle exceeds limit")
	}
	item := *input
	seen := map[string]bool{}
	var overlays []map[string]interface{}
	for item.Ref != "" {
		if depth > 0 {
			if err := r.charge(); err != nil {
				return item, err
			}
		}
		if err := budget.reserveResolution(item.Ref, item.Extensions); err != nil {
			return item, err
		}
		overlays = append(overlays, item.Extensions)
		ref := item.Ref
		if seen[ref] {
			return item, fmt.Errorf("cyclic path reference %.128q", ref)
		}
		seen[ref] = true
		// A Path Item reference with siblings has undefined behavior.
		if item.Get != nil || item.Put != nil || item.Post != nil || item.Delete != nil ||
			item.Options != nil || item.Head != nil || item.Patch != nil || item.Trace != nil ||
			len(item.Parameters) > 0 || len(item.Servers) > 0 || item.Summary != "" || item.Description != "" {
			return item, fmt.Errorf("path reference %.128q has unsupported sibling fields", ref)
		}
		var target PathItem
		var ok bool
		var err error
		switch {
		case strings.HasPrefix(ref, "#/paths/"):
			var n string
			n, err = referenceName(ref, "#/paths/")
			target, ok = doc.Paths[n]
		case strings.HasPrefix(ref, "#/webhooks/"):
			var n string
			n, err = referenceName(ref, "#/webhooks/")
			target, ok = doc.Webhooks[n]
		default:
			return item, fmt.Errorf("unsupported path reference %.128q", ref)
		}
		if err != nil {
			return item, err
		}
		if !ok {
			return item, fmt.Errorf("path reference %.128q not found", ref)
		}
		item = target
	}
	if err := budget.reserveResolution(item.Ref, item.Extensions); err != nil {
		return item, err
	}
	item.Extensions = mergeReferenceExtensions(item.Extensions, overlays)
	for i := range item.Parameters {
		p, err := resolveParameter(&item.Parameters[i], doc.Components.Parameters, "#/components/parameters/", budget)
		if err != nil {
			return item, err
		}
		item.Parameters[i] = p
	}
	for _, op := range []*Operation{item.Get, item.Post, item.Put, item.Delete, item.Options, item.Head, item.Patch, item.Trace} {
		if err := resolveOperation(op, doc.Components.Parameters, doc.Components.Responses, doc.Components.RequestBodies, false, budget); err != nil {
			return item, err
		}
		if op != nil {
			for name, callback := range op.Callbacks {
				resolved, err := r.resolveCallback(name, callback, depth+1)
				if err != nil {
					// Keep the outer callback context without rebuilding the error at each depth.
					if depth == 0 {
						return item, fmt.Errorf("callback %.128q: %w", name, err)
					}
					return item, err
				}
				op.Callbacks[name] = resolved
			}
		}
	}
	return item, nil
}

func (r *callbackResolver) resolveCallback(name string, callback Callback, depth int) (Callback, error) {
	doc := r.doc
	if err := r.charge(); err != nil {
		return Callback{}, err
	}
	if depth > 64 {
		return Callback{}, fmt.Errorf("callback nesting or reference cycle exceeds limit")
	}
	if !r.fitsModel(reflect.ValueOf(name)) {
		return Callback{}, fmt.Errorf("callback model expansion limit exceeded")
	}
	seen := map[string]bool{}
	var overlays []map[string]interface{}
	for callback.Ref != "" {
		if err := r.charge(); err != nil {
			return Callback{}, err
		}
		if err := r.reserveResolution(callback.Ref, callback.Extensions); err != nil {
			return Callback{}, err
		}
		overlays = append(overlays, callback.Extensions)
		ref := callback.Ref
		if seen[ref] {
			return Callback{}, fmt.Errorf("cyclic callback reference %.128q", ref)
		}
		seen[ref] = true
		name, err := referenceName(ref, "#/components/callbacks/")
		if err != nil {
			return Callback{}, err
		}
		var ok bool
		callback, ok = doc.Components.Callbacks[name]
		if !ok {
			return Callback{}, fmt.Errorf("callback reference %.128q not found", ref)
		}
	}
	if len(callback.Paths) > r.remaining {
		return Callback{}, fmt.Errorf("callback expansion limit exceeded")
	}
	paths := make(map[string]PathItem, len(callback.Paths))
	for expression := range callback.Paths {
		if !r.fitsModel(reflect.ValueOf(expression)) {
			return Callback{}, fmt.Errorf("callback model expansion limit exceeded")
		}
		path := callback.Paths[expression]
		resolved, err := r.resolvePath(&path, depth+1)
		if err != nil {
			if depth <= 1 {
				return Callback{}, fmt.Errorf("expression %.128q: %w", expression, err)
			}
			return Callback{}, err
		}
		paths[expression] = resolved
	}
	if err := r.reserveResolution(callback.Ref, callback.Extensions); err != nil {
		return Callback{}, err
	}
	result := Callback{Paths: paths, Extensions: mergeReferenceExtensions(callback.Extensions, overlays)}
	// Components start at depth 0, and callbacks on top-level operations at 1.
	// Count each expanded tree once, immediately before it can enter the model.
	if depth <= 1 && !r.fitsModel(reflect.ValueOf(result)) {
		return Callback{}, fmt.Errorf("callback model expansion limit exceeded")
	}
	return result, nil
}

// Charge resolved data by occurrence, including values inherited from component
// references. This bounds copies of a large schema as well as callback fan-out.
func (r *callbackResolver) fitsModel(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	r.modelBytes -= 16
	if r.modelBytes < 0 {
		return false
	}
	switch v.Kind() {
	case reflect.String:
		r.modelBytes -= int64(v.Len())
	case reflect.Interface, reflect.Pointer:
		return r.fitsModel(v.Elem())
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			if !r.fitsModel(iter.Key()) || !r.fitsModel(iter.Value()) {
				return false
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !r.fitsModel(v.Index(i)) {
				return false
			}
		}
	case reflect.Struct:
		// Wire types retain reference metadata and typed additionalProperties
		// in private fields. Conversion consumes these fields too.
		wireType := v.Type().PkgPath() == reflect.TypeFor[Callback]().PkgPath()
		for i := 0; i < v.NumField(); i++ {
			if (wireType || v.Type().Field(i).IsExported()) && !r.fitsModel(v.Field(i)) {
				return false
			}
		}
	}
	return r.modelBytes >= 0
}

func resolvePathV2(input *PathItemV2, doc *Swagger20) (PathItemV2, error) {
	item := *input
	seen := map[string]bool{}
	var overlays []map[string]interface{}
	for item.Ref != "" {
		overlays = append(overlays, item.Extensions)
		ref := item.Ref
		if seen[ref] {
			return item, fmt.Errorf("cyclic path reference %.128q", ref)
		}
		seen[ref] = true
		if item.Get != nil || item.Put != nil || item.Post != nil || item.Delete != nil ||
			item.Options != nil || item.Head != nil || item.Patch != nil || len(item.Parameters) > 0 {
			return item, fmt.Errorf("path reference %.128q has unsupported sibling fields", ref)
		}
		name, err := referenceName(ref, "#/paths/")
		if err != nil {
			return item, err
		}
		target, ok := doc.Paths[name]
		if !ok {
			return item, fmt.Errorf("path reference %.128q not found", ref)
		}
		item = target
	}
	item.Extensions = mergeReferenceExtensions(item.Extensions, overlays)
	for i := range item.Parameters {
		p, err := resolveParameter(&item.Parameters[i], doc.Parameters, "#/parameters/", nil)
		if err != nil {
			return item, err
		}
		item.Parameters[i] = p
	}
	for _, op := range []*Operation{item.Get, item.Post, item.Put, item.Delete, item.Options, item.Head, item.Patch} {
		if err := resolveOperation(op, doc.Parameters, doc.Responses, nil, true, nil); err != nil {
			return item, err
		}
	}
	return item, nil
}
