package validator

import (
	"context"
	"fmt"
	"strings"

	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

// PathValidationRule validates that paths start with a '/'.
type PathValidationRule struct{}

// SchemaValidationRule validates schema types.
type SchemaValidationRule struct{}

// SecurityValidationRule is reserved for a future rule and does not implement Rule.
type SecurityValidationRule struct{}

// OperationIDRule is reserved for a future rule and does not implement Rule.
type OperationIDRule struct{}

// Name returns the name of the PathValidationRule.
func (r *PathValidationRule) Name() string { return "path-validation" }

// Validate validates paths in the API model.
func (r *PathValidationRule) Validate(ctx context.Context, api *model.API) []format.ValidationError {
	var errors []format.ValidationError
	for path := range api.Paths {
		if !strings.HasPrefix(path, "/") {
			errors = append(errors, format.ValidationError{
				Path:    path,
				Message: "Path must start with /",
				Level:   r.Level(),
			})
		}
	}
	return errors
}

// Level returns the validation level for PathValidationRule.
func (r *PathValidationRule) Level() format.ValidationLevel { return format.LevelError }

// Name returns the name of the SchemaValidationRule.
func (r *SchemaValidationRule) Name() string { return "schema-validation" }

// Validate validates schemas in the API model.
func (r *SchemaValidationRule) Validate(ctx context.Context, api *model.API) []format.ValidationError {
	var errors []format.ValidationError
	for name, schema := range api.Components.Schemas {
		errs := ValidateSchemaNode(schema, fmt.Sprintf("components/schemas/%s", name))
		errors = append(errors, errs...)
	}
	return errors
}

// ValidateSchemaNode checks the type of a single schema node. It does not recurse.
func ValidateSchemaNode(schema *model.Schema, path string) []format.ValidationError {
	if schema == nil {
		return []format.ValidationError{{Path: path, Message: "Schema must not be null", Level: format.LevelError}}
	}
	var errors []format.ValidationError
	if schema.Type != "" {
		switch schema.Type {
		case "null", model.TypeString, model.TypeNumber, model.TypeInteger, model.TypeBoolean, model.TypeArray, model.TypeObject:
			// Valid
		default:
			errors = append(errors, format.ValidationError{
				Path:    path,
				Message: fmt.Sprintf("Invalid schema type: %s", schema.Type),
				Level:   format.LevelError,
			})
		}
	}
	return errors
}

// Level returns the validation level for SchemaValidationRule.
func (r *SchemaValidationRule) Level() format.ValidationLevel { return format.LevelError }

// ReferenceValidationRule checks local schema references in component schemas,
// path parameters, and selected operations on paths. It does not check every
// schema location or external reference.
type ReferenceValidationRule struct{}

// Name returns the name of the ReferenceValidationRule.
func (r *ReferenceValidationRule) Name() string { return "reference-validation" }

// Validate validates references in the API model.
func (r *ReferenceValidationRule) Validate(ctx context.Context, api *model.API) []format.ValidationError {
	var errors []format.ValidationError

	active := make(map[*model.Schema]bool)
	var validateSchema func(s *model.Schema, path string)
	validateSchema = func(s *model.Schema, path string) {
		if s == nil || active[s] {
			return
		}
		active[s] = true
		defer delete(active, s)
		if strings.HasPrefix(s.Ref, "#/components/schemas/") {
			if _, err := s.ResolveRef(&api.Components); err != nil {
				errors = append(errors, format.ValidationError{
					Path:    path,
					Message: fmt.Sprintf("Broken reference: %s", s.Ref),
					Level:   r.Level(),
				})
			}
		}
		validateSchema(s.Items, path+"/items")
		for k, v := range s.Properties {
			validateSchema(v, fmt.Sprintf("%s/properties/%s", path, k))
		}
	}

	for name, schema := range api.Components.Schemas {
		validateSchema(schema, fmt.Sprintf("components/schemas/%s", name))
	}

	// Check path and operation parameters and request and response bodies.
	for path := range api.Paths {
		item := api.Paths[path]
		for i := range item.Parameters {
			validateSchema(item.Parameters[i].Schema, fmt.Sprintf("paths[%s]/parameters[%d]", path, i))
		}
		checkOp := func(op *model.Operation, method string) {
			if op == nil {
				return
			}
			for i := range op.Parameters {
				validateSchema(op.Parameters[i].Schema, fmt.Sprintf("paths[%s]/%s/parameters[%d]", path, method, i))
			}
			if op.RequestBody != nil {
				for ct, mt := range op.RequestBody.Content {
					validateSchema(mt.Schema, fmt.Sprintf("paths[%s]/%s/requestBody/content[%s]", path, method, ct))
				}
			}
			for status, resp := range op.Responses {
				for ct, mt := range resp.Content {
					validateSchema(mt.Schema, fmt.Sprintf("paths[%s]/%s/responses[%s]/content[%s]", path, method, status, ct))
				}
			}
		}

		checkOp(item.Get, "get")
		checkOp(item.Post, "post")
		checkOp(item.Put, "put")
		checkOp(item.Delete, "delete")
		checkOp(item.Patch, "patch")
	}

	return errors
}

// Level returns the validation level for ReferenceValidationRule.
func (r *ReferenceValidationRule) Level() format.ValidationLevel { return format.LevelError }

// APIVersionWarningRule warns when an OpenAPI model has no API version.
type APIVersionWarningRule struct{}

// Name returns the rule name.
func (r *APIVersionWarningRule) Name() string { return "api-version" }

// Level returns the severity of the issue.
func (r *APIVersionWarningRule) Level() format.ValidationLevel { return format.LevelWarning }

// Validate reports an empty OpenAPI info.version.
func (r *APIVersionWarningRule) Validate(ctx context.Context, api *model.API) []format.ValidationError {
	if formatFromContext(ctx) == format.FormatOpenAPI && strings.TrimSpace(api.Info.Version) == "" {
		return []format.ValidationError{{Path: "info.version", Message: "API version is empty", Level: r.Level()}}
	}
	return nil
}
