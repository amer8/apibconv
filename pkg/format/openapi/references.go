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

func resolveParameter(v Parameter, values map[string]Parameter, prefix string) (Parameter, error) {
	seen := map[string]bool{}
	for v.Ref != "" {
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
	return v, nil
}
func resolveResponse(v Response, values map[string]Response, prefix string) (Response, error) {
	seen := map[string]bool{}
	for v.Ref != "" {
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
	return v, nil
}
func resolveRequestBody(v RequestBody, values map[string]RequestBody) (RequestBody, error) {
	seen := map[string]bool{}
	for v.Ref != "" {
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
	for i, v := range op.Parameters {
		p, err := resolveParameter(v, params, paramPrefix)
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
func resolvePathV3(item PathItem, doc *OpenAPI) (PathItem, error) {
	seen := map[string]bool{}
	for item.Ref != "" {
		ref := item.Ref
		if seen[ref] {
			return item, fmt.Errorf("cyclic path reference %q", ref)
		}
		seen[ref] = true
		// Siblings of a Path Item reference have undefined behavior; reject instead of silently discarding them.
		if item.Get != nil || item.Put != nil || item.Post != nil || item.Delete != nil || item.Options != nil || item.Head != nil || item.Patch != nil || item.Trace != nil || len(item.Parameters) > 0 || len(item.Servers) > 0 || item.Summary != "" || item.Description != "" {
			return item, fmt.Errorf("path reference %q has unsupported sibling fields", ref)
		}
		var target PathItem
		var ok bool
		var err error
		if strings.HasPrefix(ref, "#/paths/") {
			var n string
			n, err = referenceName(ref, "#/paths/")
			target, ok = doc.Paths[n]
		} else if strings.HasPrefix(ref, "#/webhooks/") {
			var n string
			n, err = referenceName(ref, "#/webhooks/")
			target, ok = doc.Webhooks[n]
		} else {
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
	for i, v := range item.Parameters {
		p, err := resolveParameter(v, doc.Components.Parameters, "#/components/parameters/")
		if err != nil {
			return item, err
		}
		item.Parameters[i] = p
	}
	for _, op := range []*Operation{item.Get, item.Post, item.Put, item.Delete, item.Options, item.Head, item.Patch, item.Trace} {
		if err := resolveOperation(op, doc.Components.Parameters, doc.Components.Responses, doc.Components.RequestBodies, false); err != nil {
			return item, err
		}
	}
	return item, nil
}
func resolvePathV2(item PathItemV2, doc *Swagger20) (PathItemV2, error) {
	seen := map[string]bool{}
	for item.Ref != "" {
		ref := item.Ref
		if seen[ref] {
			return item, fmt.Errorf("cyclic path reference %q", ref)
		}
		seen[ref] = true
		if item.Get != nil || item.Put != nil || item.Post != nil || item.Delete != nil || item.Options != nil || item.Head != nil || item.Patch != nil || len(item.Parameters) > 0 {
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
	for i, v := range item.Parameters {
		p, err := resolveParameter(v, doc.Parameters, "#/parameters/")
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
