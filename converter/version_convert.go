package converter

import (
	"fmt"
)

// ConvertToVersion converts an OpenAPI specification to a target version.
//
// This function handles bidirectional conversion between OpenAPI 3.0 and 3.1.
// When converting from 3.1 to 3.0, features that don't exist in 3.0 (like webhooks)
// are dropped unless StrictMode is enabled in the options.
//
// Parameters:
//   - spec: The OpenAPI specification to convert
//   - targetVersion: The desired output version (Version30 or Version31)
//   - opts: Optional conversion options (can be nil for defaults)
//
// Returns:
//   - *OpenAPI: The converted specification
//   - error: Error if conversion fails
//
// Example:
//
//	spec, _ := converter.Parse(jsonData)
//	converted, err := converter.ConvertToVersion(spec, converter.Version31, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
func ConvertToVersion(spec *OpenAPI, targetVersion Version, opts *ConversionOptions) (*OpenAPI, error) {
	if spec == nil {
		return nil, fmt.Errorf("spec cannot be nil")
	}

	if opts == nil {
		opts = DefaultConversionOptions()
	}

	currentVersion := DetectVersion(spec.OpenAPI)

	// No conversion needed
	if currentVersion == targetVersion {
		return spec, nil
	}

	// Create a deep copy to avoid modifying the original
	converted := copyOpenAPI(spec)

	switch {
	case currentVersion == Version30 && targetVersion == Version31:
		return convert30To31(converted, opts)
	case currentVersion == Version31 && targetVersion == Version30:
		return convert31To30(converted, opts)
	default:
		return nil, fmt.Errorf("unsupported conversion from %s to %s", currentVersion, targetVersion)
	}
}

// convert30To31 upgrades an OpenAPI 3.0 specification to 3.1.
//
// This conversion:
//   - Updates the version string to "3.1.0"
//   - Converts nullable: true to type arrays (e.g., ["string", "null"])
//   - Preserves all existing features (3.0 is a subset of 3.1)
func convert30To31(spec *OpenAPI, opts *ConversionOptions) (*OpenAPI, error) {
	spec.OpenAPI = "3.1.0"

	// Convert all schemas from 3.0 style (nullable) to 3.1 style (type arrays)
	if spec.Components != nil && spec.Components.Schemas != nil {
		for _, schema := range spec.Components.Schemas {
			convertSchemaToV31(schema)
		}
	}

	// Convert schemas in paths
	for k, pathItem := range spec.Paths {
		item := pathItem // Create a copy to avoid implicit memory aliasing
		convertPathItemSchemasToV31(&item)
		spec.Paths[k] = item
	}

	return spec, nil
}

// convert31To30 downgrades an OpenAPI 3.1 specification to 3.0.
//
// This conversion:
//   - Updates the version string to "3.0.0"
//   - Converts type arrays to nullable: true
//   - Removes 3.1-only features (webhooks, jsonSchemaDialect, etc.)
//   - In StrictMode, fails if 3.1-only features are present
func convert31To30(spec *OpenAPI, opts *ConversionOptions) (*OpenAPI, error) {
	// Check for 3.1-only features in strict mode
	if opts.StrictMode {
		if len(spec.Webhooks) > 0 {
			return nil, fmt.Errorf("webhooks are not supported in OpenAPI 3.0 (use StrictMode: false to drop them)")
		}
		if spec.JSONSchemaDialect != "" {
			return nil, fmt.Errorf("jsonSchemaDialect is not supported in OpenAPI 3.0")
		}
		if spec.Info.License != nil && spec.Info.License.Identifier != "" {
			return nil, fmt.Errorf("license.identifier (SPDX) is not supported in OpenAPI 3.0")
		}
	}

	spec.OpenAPI = "3.0.0"

	// Remove 3.1-only features
	spec.Webhooks = nil
	spec.JSONSchemaDialect = ""

	if spec.Info.License != nil {
		spec.Info.License.Identifier = ""
	}

	// Convert all schemas from 3.1 style (type arrays) to 3.0 style (nullable)
	if spec.Components != nil && spec.Components.Schemas != nil {
		for _, schema := range spec.Components.Schemas {
			convertSchemaToV30(schema)
		}
	}

	// Convert schemas in paths
	for k, pathItem := range spec.Paths {
		item := pathItem // Create a copy to avoid implicit memory aliasing
		convertPathItemSchemasToV30(&item)
		spec.Paths[k] = item
	}

	return spec, nil
}

// convertSchemaToV31 converts a schema from OpenAPI 3.0 to 3.1 format.
//
// This primarily involves converting nullable: true to type arrays.
// For example, {type: "string", nullable: true} becomes {type: ["string", "null"]}.
func convertSchemaToV31(schema *Schema) {
	if schema == nil {
		return
	}

	// Convert nullable to type array
	if schema.Nullable && schema.Type != nil {
		// Check if Type is a string
		if typeStr, ok := schema.Type.(string); ok && typeStr != "" {
			schema.Type = []interface{}{typeStr, "null"}
			schema.Nullable = false
		}
	}

	// Recursively convert nested schemas
	for _, prop := range schema.Properties {
		convertSchemaToV31(prop)
	}

	if schema.Items != nil {
		convertSchemaToV31(schema.Items)
	}

	// Convert dependent schemas
	for _, depSchema := range schema.DependentSchemas {
		convertSchemaToV31(depSchema)
	}

	// Convert prefix items
	for _, prefixItem := range schema.PrefixItems {
		convertSchemaToV31(prefixItem)
	}
}

// convertSchemaToV30 converts a schema from OpenAPI 3.1 to 3.0 format.
//
// This primarily involves converting type arrays to nullable: true.
// For example, {type: ["string", "null"]} becomes {type: "string", nullable: true}.
func convertSchemaToV30(schema *Schema) {
	if schema == nil {
		return
	}

	// Convert type array to nullable
	if schema.Type != nil {
		// Check if Type is an array (slice)
		if typeArray, ok := schema.Type.([]interface{}); ok {
			// Look for ["type", "null"] pattern
			var mainType string
			hasNull := false

			for _, t := range typeArray {
				if tStr, ok := t.(string); ok {
					if tStr == "null" {
						hasNull = true
					} else {
						mainType = tStr
					}
				}
			}

			if hasNull && mainType != "" {
				schema.Type = mainType
				schema.Nullable = true
			} else if len(typeArray) == 1 {
				// Single-element array, just use the type
				if tStr, ok := typeArray[0].(string); ok {
					schema.Type = tStr
				}
			}
		}
	}

	// Remove 3.1-only fields
	schema.Const = nil
	schema.DependentSchemas = nil
	schema.PrefixItems = nil

	// Recursively convert nested schemas
	for _, prop := range schema.Properties {
		convertSchemaToV30(prop)
	}

	if schema.Items != nil {
		convertSchemaToV30(schema.Items)
	}
}

// convertPathItemSchemasToV31 converts all schemas within a PathItem to 3.1 format.
func convertPathItemSchemasToV31(pathItem *PathItem) {
	if pathItem.Get != nil {
		convertOperationSchemasToV31(pathItem.Get)
	}
	if pathItem.Post != nil {
		convertOperationSchemasToV31(pathItem.Post)
	}
	if pathItem.Put != nil {
		convertOperationSchemasToV31(pathItem.Put)
	}
	if pathItem.Delete != nil {
		convertOperationSchemasToV31(pathItem.Delete)
	}
	if pathItem.Patch != nil {
		convertOperationSchemasToV31(pathItem.Patch)
	}
}

// convertPathItemSchemasToV30 converts all schemas within a PathItem to 3.0 format.
func convertPathItemSchemasToV30(pathItem *PathItem) {
	if pathItem.Get != nil {
		convertOperationSchemasToV30(pathItem.Get)
	}
	if pathItem.Post != nil {
		convertOperationSchemasToV30(pathItem.Post)
	}
	if pathItem.Put != nil {
		convertOperationSchemasToV30(pathItem.Put)
	}
	if pathItem.Delete != nil {
		convertOperationSchemasToV30(pathItem.Delete)
	}
	if pathItem.Patch != nil {
		convertOperationSchemasToV30(pathItem.Patch)
	}
}

// convertOperationSchemasToV31 converts all schemas within an Operation to 3.1 format.
func convertOperationSchemasToV31(op *Operation) {
	// Convert parameter schemas
	for i := range op.Parameters {
		if op.Parameters[i].Schema != nil {
			convertSchemaToV31(op.Parameters[i].Schema)
		}
	}

	// Convert request body schemas
	if op.RequestBody != nil {
		for _, mediaType := range op.RequestBody.Content {
			if mediaType.Schema != nil {
				convertSchemaToV31(mediaType.Schema)
			}
		}
	}

	// Convert response schemas
	for _, response := range op.Responses {
		for _, mediaType := range response.Content {
			if mediaType.Schema != nil {
				convertSchemaToV31(mediaType.Schema)
			}
		}
	}
}

// convertOperationSchemasToV30 converts all schemas within an Operation to 3.0 format.
func convertOperationSchemasToV30(op *Operation) {
	// Convert parameter schemas
	for i := range op.Parameters {
		if op.Parameters[i].Schema != nil {
			convertSchemaToV30(op.Parameters[i].Schema)
		}
	}

	// Convert request body schemas
	if op.RequestBody != nil {
		for _, mediaType := range op.RequestBody.Content {
			if mediaType.Schema != nil {
				convertSchemaToV30(mediaType.Schema)
			}
		}
	}

	// Convert response schemas
	for _, response := range op.Responses {
		for _, mediaType := range response.Content {
			if mediaType.Schema != nil {
				convertSchemaToV30(mediaType.Schema)
			}
		}
	}
}

// copyOpenAPI creates a deep copy of an OpenAPI spec to prevent mutations.
// This ensures that schema conversions don't affect the original spec.
func copyOpenAPI(spec *OpenAPI) *OpenAPI {
	if spec == nil {
		return nil
	}

	copied := &OpenAPI{
		OpenAPI:           spec.OpenAPI,
		Info:              spec.Info,
		Servers:           append([]Server{}, spec.Servers...),
		JSONSchemaDialect: spec.JSONSchemaDialect,
	}

	// Deep copy paths with schemas
	copied.Paths = make(map[string]PathItem, len(spec.Paths))
	for k, v := range spec.Paths {
		pathItem := v // Create a copy to avoid implicit memory aliasing
		copied.Paths[k] = deepCopyPathItem(&pathItem)
	}

	// Deep copy webhooks with schemas
	if spec.Webhooks != nil {
		copied.Webhooks = make(map[string]PathItem, len(spec.Webhooks))
		for k, v := range spec.Webhooks {
			pathItem := v // Create a copy to avoid implicit memory aliasing
			copied.Webhooks[k] = deepCopyPathItem(&pathItem)
		}
	}

	// Deep copy components with schemas
	if spec.Components != nil {
		copied.Components = &Components{}
		if spec.Components.Schemas != nil {
			copied.Components.Schemas = make(map[string]*Schema, len(spec.Components.Schemas))
			for k, v := range spec.Components.Schemas {
				copied.Components.Schemas[k] = deepCopySchema(v)
			}
		}
	}

	// Copy license if present
	if spec.Info.License != nil {
		licenseCopy := *spec.Info.License
		copied.Info.License = &licenseCopy
	}

	return copied
}

// deepCopyPathItem creates a deep copy of a PathItem, including all operations and schemas.
func deepCopyPathItem(pathItem *PathItem) PathItem {
	if pathItem == nil {
		return PathItem{}
	}

	copied := PathItem{}

	if pathItem.Get != nil {
		op := deepCopyOperation(pathItem.Get)
		copied.Get = &op
	}
	if pathItem.Post != nil {
		op := deepCopyOperation(pathItem.Post)
		copied.Post = &op
	}
	if pathItem.Put != nil {
		op := deepCopyOperation(pathItem.Put)
		copied.Put = &op
	}
	if pathItem.Delete != nil {
		op := deepCopyOperation(pathItem.Delete)
		copied.Delete = &op
	}
	if pathItem.Patch != nil {
		op := deepCopyOperation(pathItem.Patch)
		copied.Patch = &op
	}

	return copied
}

// deepCopyOperation creates a deep copy of an Operation, including all schemas.
func deepCopyOperation(op *Operation) Operation {
	if op == nil {
		return Operation{}
	}

	copied := Operation{
		Summary:     op.Summary,
		Description: op.Description,
	}

	// Deep copy parameters with schemas
	if len(op.Parameters) > 0 {
		copied.Parameters = make([]Parameter, len(op.Parameters))
		for i, param := range op.Parameters {
			copied.Parameters[i] = Parameter{
				Name:        param.Name,
				In:          param.In,
				Required:    param.Required,
				Description: param.Description,
			}
			if param.Schema != nil {
				copied.Parameters[i].Schema = deepCopySchema(param.Schema)
			}
		}
	}

	// Deep copy request body
	if op.RequestBody != nil {
		copied.RequestBody = &RequestBody{
			Description: op.RequestBody.Description,
			Required:    op.RequestBody.Required,
			Content:     make(map[string]MediaType, len(op.RequestBody.Content)),
		}
		for mediaType, mt := range op.RequestBody.Content {
			copied.RequestBody.Content[mediaType] = MediaType{
				Schema: deepCopySchema(mt.Schema),
			}
		}
	}

	// Deep copy responses
	if len(op.Responses) > 0 {
		copied.Responses = make(map[string]Response, len(op.Responses))
		for code, resp := range op.Responses {
			copiedResp := Response{
				Description: resp.Description,
				Content:     make(map[string]MediaType, len(resp.Content)),
			}
			for mediaType, mt := range resp.Content {
				copiedResp.Content[mediaType] = MediaType{
					Schema: deepCopySchema(mt.Schema),
				}
			}
			copied.Responses[code] = copiedResp
		}
	}

	return copied
}

// deepCopySchema creates a deep copy of a Schema to prevent mutations.
func deepCopySchema(schema *Schema) *Schema {
	if schema == nil {
		return nil
	}

	copied := &Schema{
		Type:     schema.Type, // interface{} - will be copied by value
		Nullable: schema.Nullable,
		Example:  schema.Example,
		Const:    schema.Const,
	}

	// Deep copy properties
	if len(schema.Properties) > 0 {
		copied.Properties = make(map[string]*Schema, len(schema.Properties))
		for k, v := range schema.Properties {
			copied.Properties[k] = deepCopySchema(v)
		}
	}

	// Deep copy items
	if schema.Items != nil {
		copied.Items = deepCopySchema(schema.Items)
	}

	// Deep copy dependent schemas (3.1 feature)
	if len(schema.DependentSchemas) > 0 {
		copied.DependentSchemas = make(map[string]*Schema, len(schema.DependentSchemas))
		for k, v := range schema.DependentSchemas {
			copied.DependentSchemas[k] = deepCopySchema(v)
		}
	}

	// Deep copy prefix items (3.1 feature)
	if len(schema.PrefixItems) > 0 {
		copied.PrefixItems = make([]*Schema, len(schema.PrefixItems))
		for i, v := range schema.PrefixItems {
			copied.PrefixItems[i] = deepCopySchema(v)
		}
	}

	return copied
}

// GetSchemaType returns the primary type of a schema as a string.
//
// In OpenAPI 3.0, Type is always a string.
// In OpenAPI 3.1, Type can be a string or []string.
// This helper extracts the primary (non-null) type.
func GetSchemaType(schema *Schema) string {
	if schema == nil || schema.Type == nil {
		return ""
	}

	// Handle string type
	if typeStr, ok := schema.Type.(string); ok {
		return typeStr
	}

	// Handle array type (3.1)
	if typeArr, ok := schema.Type.([]interface{}); ok {
		for _, t := range typeArr {
			if tStr, ok := t.(string); ok && tStr != "null" {
				return tStr
			}
		}
	}

	return ""
}

// IsNullable returns true if a schema allows null values.
//
// In OpenAPI 3.0, this is indicated by nullable: true.
// In OpenAPI 3.1, this is indicated by type: [..., "null"].
func IsNullable(schema *Schema) bool {
	if schema == nil {
		return false
	}

	// Check 3.0 style nullable
	if schema.Nullable {
		return true
	}

	// Check 3.1 style type array
	if typeArr, ok := schema.Type.([]interface{}); ok {
		for _, t := range typeArr {
			if tStr, ok := t.(string); ok && tStr == "null" {
				return true
			}
		}
	}

	return false
}

// NormalizeSchemaType ensures schema.Type is properly formatted for JSON marshaling.
//
// This is useful when you've manipulated the Type field and want to ensure
// it's in the correct format for the target OpenAPI version.
func NormalizeSchemaType(schema *Schema, version Version) {
	if schema == nil || schema.Type == nil {
		return
	}

	// For 3.0, ensure Type is a string
	if version == Version30 {
		typeStr := GetSchemaType(schema)
		if typeStr != "" {
			schema.Type = typeStr
		}
	}

	// For 3.1, convert to array if nullable
	if version == Version31 && schema.Nullable {
		typeStr := GetSchemaType(schema)
		if typeStr != "" {
			schema.Type = []interface{}{typeStr, "null"}
			schema.Nullable = false
		}
	}

	// Recursively normalize nested schemas
	for _, prop := range schema.Properties {
		NormalizeSchemaType(prop, version)
	}

	if schema.Items != nil {
		NormalizeSchemaType(schema.Items, version)
	}
}
