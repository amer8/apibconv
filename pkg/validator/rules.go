package validator

import (
	"fmt"
	"strings"

	"github.com/amer8/apibconv/pkg/format"
	"github.com/amer8/apibconv/pkg/model"
)

// PathValidationRule validates that paths start with a '/'.
type PathValidationRule struct{}

// SchemaValidationRule validates schema types.
type SchemaValidationRule struct{}

// SecurityValidationRule (Not yet implemented) validates security schemes.
type SecurityValidationRule struct{}

// OperationIDRule (Not yet implemented) validates operation IDs.
type OperationIDRule struct{}

// Name returns the name of the PathValidationRule.
func (r *PathValidationRule) Name() string { return "path-validation" }

// Validate validates paths in the API model.
func (r *PathValidationRule) Validate(api *model.API) []format.ValidationError {
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
func (r *SchemaValidationRule) Validate(api *model.API) []format.ValidationError {
	var errors []format.ValidationError
	for name, schema := range api.Components.Schemas {
		if schema.Type != "" {
			switch schema.Type {
			case model.TypeString, model.TypeNumber, model.TypeInteger, model.TypeBoolean, model.TypeArray, model.TypeObject:
				// Valid
			default:
				errors = append(errors, format.ValidationError{
					Path:    fmt.Sprintf("components/schemas/%s", name),
					Message: fmt.Sprintf("Invalid schema type: %s", schema.Type),
					Level:   r.Level(),
				})
			}
		}
	}
	return errors
}

// Level returns the validation level for SchemaValidationRule.
func (r *SchemaValidationRule) Level() format.ValidationLevel { return format.LevelError }

// ReferenceValidationRule validates internal references ($ref) within the API model.
type ReferenceValidationRule struct{}

// Name returns the name of the ReferenceValidationRule.
func (r *ReferenceValidationRule) Name() string { return "reference-validation" }

// Validate validates references in the API model.
func (r *ReferenceValidationRule) Validate(api *model.API) []format.ValidationError {
	var errors []format.ValidationError
	
	// Set of valid schema names
	validSchemas := make(map[string]bool)
	for name := range api.Components.Schemas {
		validSchemas[name] = true
	}

	// Helper to validate a schema's refs
	var validateSchema func(s *model.Schema, path string)
	validateSchema = func(s *model.Schema, path string) {
		if s == nil {
			return
		}
		if s.Ref != "" {
			// Expect local refs like "#/components/schemas/Name"
			if strings.HasPrefix(s.Ref, "#/components/schemas/") {
				refName := strings.TrimPrefix(s.Ref, "#/components/schemas/")
				if !validSchemas[refName] {
					errors = append(errors, format.ValidationError{
						Path:    path,
						Message: fmt.Sprintf("Broken reference: %s", s.Ref),
						Level:   r.Level(),
					})
				}
			}
		}
		if s.Items != nil {
			validateSchema(s.Items, path+"/items")
		}
		for k, v := range s.Properties {
			validateSchema(v, fmt.Sprintf("%s/properties/%s", path, k))
		}
	}

	// Check schemas in Components
	for name, schema := range api.Components.Schemas {
		validateSchema(schema, fmt.Sprintf("components/schemas/%s", name))
	}

	// Check schemas in Paths (Simplified: checking parameters and request/response bodies)
	for path := range api.Paths {
		item := api.Paths[path]
		// Check Parameters
		for i, p := range item.Parameters {
			validateSchema(p.Schema, fmt.Sprintf("paths[%s]/parameters[%d]", path, i))
		}
		// Helper to check operation
		checkOp := func(op *model.Operation, method string) {
			if op == nil { return }
			for i, p := range op.Parameters {
				validateSchema(p.Schema, fmt.Sprintf("paths[%s]/%s/parameters[%d]", path, method, i))
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