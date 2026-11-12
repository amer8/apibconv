package openapi

import (
	"fmt"
	"net/url"
	"strings"
)

// referenceName resolves a single JSON Pointer token in a local component map.
func referenceName(ref, prefix string) (string, error) {
	if !strings.HasPrefix(ref, prefix) {
		return "", fmt.Errorf("unsupported reference %q", ref)
	}
	token := strings.TrimPrefix(ref, prefix)
	if strings.Contains(token, "/") {
		return "", fmt.Errorf("unsupported reference %q", ref)
	}
	decoded, err := url.PathUnescape(token)
	if err != nil {
		return "", fmt.Errorf("invalid reference %q: %w", ref, err)
	}
	for i := 0; i < len(decoded); i++ {
		if decoded[i] == '~' {
			if i+1 == len(decoded) || (decoded[i+1] != '0' && decoded[i+1] != '1') {
				return "", fmt.Errorf("invalid JSON pointer %q", ref)
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

func resolveParameter(input *Parameter, values map[string]Parameter, prefix string) (Parameter, error) {
	v := *input
	seen := map[string]bool{}
	var overlays []map[string]interface{}
	for v.Ref != "" {
		overlays = append(overlays, v.Extensions)
		if seen[v.Ref] {
			return Parameter{}, fmt.Errorf("cyclic parameter reference %q", v.Ref)
		}
		seen[v.Ref] = true
		name, err := referenceName(v.Ref, prefix)
		if err != nil {
			return Parameter{}, err
		}
		target, ok := values[name]
		if !ok {
			return Parameter{}, fmt.Errorf("parameter reference %q not found", v.Ref)
		}
		v = target
	}
	v.Extensions = mergeReferenceExtensions(v.Extensions, overlays)
	return v, nil
}

func resolveResponse(v Response, values map[string]Response, prefix string) (Response, error) {
	seen := map[string]bool{}
	var overlays []map[string]interface{}
	for v.Ref != "" {
		overlays = append(overlays, v.Extensions)
		if seen[v.Ref] {
			return Response{}, fmt.Errorf("cyclic response reference %q", v.Ref)
		}
		seen[v.Ref] = true
		name, err := referenceName(v.Ref, prefix)
		if err != nil {
			return Response{}, err
		}
		target, ok := values[name]
		if !ok {
			return Response{}, fmt.Errorf("response reference %q not found", v.Ref)
		}
		v = target
	}
	v.Extensions = mergeReferenceExtensions(v.Extensions, overlays)
	return v, nil
}

func resolveRequestBody(v RequestBody, values map[string]RequestBody) (RequestBody, error) {
	seen := map[string]bool{}
	var overlays []map[string]interface{}
	for v.Ref != "" {
		overlays = append(overlays, v.Extensions)
		if seen[v.Ref] {
			return RequestBody{}, fmt.Errorf("cyclic request body reference %q", v.Ref)
		}
		seen[v.Ref] = true
		name, err := referenceName(v.Ref, "#/components/requestBodies/")
		if err != nil {
			return RequestBody{}, err
		}
		target, ok := values[name]
		if !ok {
			return RequestBody{}, fmt.Errorf("request body reference %q not found", v.Ref)
		}
		v = target
	}
	v.Extensions = mergeReferenceExtensions(v.Extensions, overlays)
	return v, nil
}

func resolveOperation(op *Operation, params map[string]Parameter, responses map[string]Response, bodies map[string]RequestBody, v2 bool) error {
	if op == nil {
		return nil
	}
	paramPrefix, respPrefix := "#/components/parameters/", "#/components/responses/"
	if v2 {
		paramPrefix, respPrefix = "#/parameters/", "#/responses/"
	}
	for i := range op.Parameters {
		p, err := resolveParameter(&op.Parameters[i], params, paramPrefix)
		if err != nil {
			return err
		}
		op.Parameters[i] = p
	}
	if op.RequestBody != nil {
		b, err := resolveRequestBody(*op.RequestBody, bodies)
		if err != nil {
			return err
		}
		op.RequestBody = &b
	}
	for k, v := range op.Responses {
		r, err := resolveResponse(v, responses, respPrefix)
		if err != nil {
			return err
		}
		op.Responses[k] = r
	}
	return nil
}

func resolvePathV3(input *PathItem, doc *OpenAPI) (PathItem, error) {
	return resolvePathV3Depth(input, doc, 0)
}

func resolvePathV3Depth(input *PathItem, doc *OpenAPI, depth int) (PathItem, error) {
	if depth > 64 {
		return PathItem{}, fmt.Errorf("callback nesting or reference cycle exceeds limit")
	}
	item := *input
	seen := map[string]bool{}
	var overlays []map[string]interface{}
	for item.Ref != "" {
		overlays = append(overlays, item.Extensions)
		ref := item.Ref
		if seen[ref] {
			return item, fmt.Errorf("cyclic path reference %q", ref)
		}
		seen[ref] = true
		// A Path Item reference with siblings has undefined behavior.
		if item.Get != nil || item.Put != nil || item.Post != nil || item.Delete != nil ||
			item.Options != nil || item.Head != nil || item.Patch != nil || item.Trace != nil ||
			len(item.Parameters) > 0 || len(item.Servers) > 0 || item.Summary != "" || item.Description != "" {
			return item, fmt.Errorf("path reference %q has unsupported sibling fields", ref)
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
			return item, fmt.Errorf("unsupported path reference %q", ref)
		}
		if err != nil {
			return item, err
		}
		if !ok {
			return item, fmt.Errorf("path reference %q not found", ref)
		}
		item = target
	}
	item.Extensions = mergeReferenceExtensions(item.Extensions, overlays)
	for i := range item.Parameters {
		p, err := resolveParameter(&item.Parameters[i], doc.Components.Parameters, "#/components/parameters/")
		if err != nil {
			return item, err
		}
		item.Parameters[i] = p
	}
	for _, op := range []*Operation{item.Get, item.Post, item.Put, item.Delete, item.Options, item.Head, item.Patch, item.Trace} {
		if err := resolveOperation(op, doc.Components.Parameters, doc.Components.Responses, doc.Components.RequestBodies, false); err != nil {
			return item, err
		}
		if op != nil {
			for name, callback := range op.Callbacks {
				resolved, err := resolveCallback(callback, doc, depth+1)
				if err != nil {
					return item, fmt.Errorf("callback %q: %w", name, err)
				}
				op.Callbacks[name] = resolved
			}
		}
	}
	return item, nil
}

func resolveCallback(callback Callback, doc *OpenAPI, depth int) (Callback, error) {
	if depth > 64 {
		return Callback{}, fmt.Errorf("callback nesting or reference cycle exceeds limit")
	}
	seen := map[string]bool{}
	var overlays []map[string]interface{}
	for callback.Ref != "" {
		overlays = append(overlays, callback.Extensions)
		ref := callback.Ref
		if seen[ref] {
			return Callback{}, fmt.Errorf("cyclic callback reference %q", ref)
		}
		seen[ref] = true
		name, err := referenceName(ref, "#/components/callbacks/")
		if err != nil {
			return Callback{}, err
		}
		var ok bool
		callback, ok = doc.Components.Callbacks[name]
		if !ok {
			return Callback{}, fmt.Errorf("callback reference %q not found", ref)
		}
	}
	paths := make(map[string]PathItem, len(callback.Paths))
	for expression := range callback.Paths {
		path := callback.Paths[expression]
		resolved, err := resolvePathV3Depth(&path, doc, depth+1)
		if err != nil {
			return Callback{}, fmt.Errorf("expression %q: %w", expression, err)
		}
		paths[expression] = resolved
	}
	return Callback{Paths: paths, Extensions: mergeReferenceExtensions(callback.Extensions, overlays)}, nil
}

func resolvePathV2(input *PathItemV2, doc *Swagger20) (PathItemV2, error) {
	item := *input
	seen := map[string]bool{}
	var overlays []map[string]interface{}
	for item.Ref != "" {
		overlays = append(overlays, item.Extensions)
		ref := item.Ref
		if seen[ref] {
			return item, fmt.Errorf("cyclic path reference %q", ref)
		}
		seen[ref] = true
		if item.Get != nil || item.Put != nil || item.Post != nil || item.Delete != nil ||
			item.Options != nil || item.Head != nil || item.Patch != nil || len(item.Parameters) > 0 {
			return item, fmt.Errorf("path reference %q has unsupported sibling fields", ref)
		}
		name, err := referenceName(ref, "#/paths/")
		if err != nil {
			return item, err
		}
		target, ok := doc.Paths[name]
		if !ok {
			return item, fmt.Errorf("path reference %q not found", ref)
		}
		item = target
	}
	item.Extensions = mergeReferenceExtensions(item.Extensions, overlays)
	for i := range item.Parameters {
		p, err := resolveParameter(&item.Parameters[i], doc.Parameters, "#/parameters/")
		if err != nil {
			return item, err
		}
		item.Parameters[i] = p
	}
	for _, op := range []*Operation{item.Get, item.Post, item.Put, item.Delete, item.Options, item.Head, item.Patch} {
		if err := resolveOperation(op, doc.Parameters, doc.Responses, nil, true); err != nil {
			return item, err
		}
	}
	return item, nil
}
