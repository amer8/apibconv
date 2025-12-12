package converter

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// Helper function to convert OpenAPI JSON to Blueprint for testing
func convertOpenAPIToBlueprint(t *testing.T, input string) string {
	t.Helper()
	var output bytes.Buffer
	s, err := Parse([]byte(input), FormatOpenAPI)
	if err == nil {
		if openapiSpec, ok := s.AsOpenAPI(); ok {
			_, err = openapiSpec.WriteTo(&output)
		} else {
			err = fmt.Errorf("spec is not an OpenAPI spec")
		}
	}
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	return output.String()
}

func TestConvert_BasicSpec(t *testing.T) {
	input := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"description": "A test API",
			"version": "1.0.0"
		},
		"servers": [
			{
				"url": "https://api.example.com"
			}
		],
		"paths": {
			"/users": {
				"get": {
					"summary": "Get users",
					"description": "Retrieve all users",
					"responses": {
						"200": {
							"description": "Successful response"
						}
					}
				}
			}
		}
	}`

	result := convertOpenAPIToBlueprint(t, input)

	// Check for required elements
	if !strings.Contains(result, "FORMAT: 1A") {
		t.Error("Missing FORMAT declaration")
	}
	if !strings.Contains(result, "# Test API") {
		t.Error("Missing API title")
	}
	if !strings.Contains(result, "HOST: https://api.example.com") {
		t.Error("Missing HOST declaration")
	}
	if !strings.Contains(result, "Get users") {
		t.Error("Missing operation summary")
	}
	if !strings.Contains(result, "Response 200") {
		t.Error("Missing response")
	}
}

func TestConvert_WithParameters(t *testing.T) {
	input := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/users/{id}": {
				"get": {
					"summary": "Get user by ID",
					"parameters": [
						{
							"name": "id",
							"in": "path",
							"description": "User ID",
							"required": true,
							"schema": {
								"type": "integer"
							}
						}
					],
					"responses": {
						"200": {
							"description": "Success"
						}
					}
				}
			}
		}
	}`

	result := convertOpenAPIToBlueprint(t, input)

	if !strings.Contains(result, "+ Parameters") {
		t.Error("Missing Parameters section")
	}
	if !strings.Contains(result, "id") {
		t.Error("Missing parameter name")
	}
	if !strings.Contains(result, "required") {
		t.Error("Missing required indicator")
	}
	if !strings.Contains(result, "integer") {
		t.Error("Missing parameter type")
	}
}

func TestConvert_WithRequestBody(t *testing.T) {
	input := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/users": {
				"post": {
					"summary": "Create user",
					"requestBody": {
						"required": true,
						"content": {
							"application/json": {
								"example": {
									"name": "John Doe",
									"email": "john@example.com"
								}
							}
						}
					},
					"responses": {
						"201": {
							"description": "Created"
						}
					}
				}
			}
		}
	}`

	result := convertOpenAPIToBlueprint(t, input)

	if !strings.Contains(result, "+ Request (application/json)") {
		t.Error("Missing Request section")
	}
	if !strings.Contains(result, "John Doe") {
		t.Error("Missing request example")
	}
}

func TestConvert_WithResponseExample(t *testing.T) {
	input := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/users": {
				"get": {
					"summary": "Get users",
					"responses": {
						"200": {
							"description": "Success",
							"content": {
								"application/json": {
									"example": {
										"users": [
											{"id": 1, "name": "Alice"},
											{"id": 2, "name": "Bob"}
										]
									}
								}
							}
						}
					}
				}
			}
		}
	}`

	result := convertOpenAPIToBlueprint(t, input)

	if !strings.Contains(result, "+ Response 200 (application/json)") {
		t.Error("Missing Response with content type")
	}
	if !strings.Contains(result, "Alice") {
		t.Error("Missing response example")
	}
}

func TestConvert_MultipleHTTPMethods(t *testing.T) {
	input := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/users": {
				"get": {
					"summary": "Get users",
					"responses": {
						"200": {
							"description": "Success"
						}
					}
				},
				"post": {
					"summary": "Create user",
					"responses": {
						"201": {
							"description": "Created"
						}
					}
				},
				"delete": {
					"summary": "Delete user",
					"responses": {
						"204": {
							"description": "Deleted"
						}
					}
				}
			}
		}
	}`

	result := convertOpenAPIToBlueprint(t, input)

	if !strings.Contains(result, "[GET]") {
		t.Error("Missing GET method")
	}
	if !strings.Contains(result, "[POST]") {
		t.Error("Missing POST method")
	}
	if !strings.Contains(result, "[DELETE]") {
		t.Error("Missing DELETE method")
	}
}

func TestConvert_InvalidJSON(t *testing.T) {
	input := `{invalid json`

	var output bytes.Buffer
	s, err := Parse([]byte(input), FormatOpenAPI)
	if err == nil {
		if openapiSpec, ok := s.AsOpenAPI(); ok {
			_, err = openapiSpec.WriteTo(&output)
		} else {
			err = fmt.Errorf("spec is not an OpenAPI spec")
		}
	}
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestConvert_EmptySpec(t *testing.T) {
	input := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Empty API",
			"version": "1.0.0"
		},
		"paths": {}
	}`

	result := convertOpenAPIToBlueprint(t, input)
	if !strings.Contains(result, "# Empty API") {
		t.Error("Missing API title")
	}
}

func TestBufferPool(t *testing.T) {
	// Test that buffer pool works correctly
	buf1 := getBuffer()
	buf1.WriteString("test")
	putBuffer(buf1)

	buf2 := getBuffer()
	if buf2.Len() != 0 {
		t.Error("Buffer should be reset when retrieved from pool")
	}
	putBuffer(buf2)
}

func TestConvert_WithRequestBodyNoExample(t *testing.T) {
	input := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/users": {
				"post": {
					"summary": "Create user",
					"requestBody": {
						"required": true,
						"content": {
							"application/json": {}
						}
					},
					"responses": {
						"201": {
							"description": "Created"
						}
					}
				}
			}
		}
	}`

	result := convertOpenAPIToBlueprint(t, input)
	if !strings.Contains(result, "+ Request (application/json)") {
		t.Error("Missing Request section")
	}
}

func TestConvert_WithRequestBodyNotRequired(t *testing.T) {
	input := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/data": {
				"put": {
					"summary": "Update data",
					"requestBody": {
						"required": false,
						"content": {
							"application/json": {
								"example": {"field": "value"}
							}
						}
					},
					"responses": {
						"200": {"description": "Updated"}
					}
				}
			}
		}
	}`

	result := convertOpenAPIToBlueprint(t, input)
	if !strings.Contains(result, "+ Request (application/json)") {
		t.Error("Missing Request section")
	}
}

func TestConvert_WithMultipleContentTypes(t *testing.T) {
	input := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/files": {
				"post": {
					"summary": "Upload file",
					"requestBody": {
						"content": {
							"multipart/form-data": {},
							"application/json": {
								"example": {"data": "test"}
							}
						}
					},
					"responses": {
						"200": {
							"description": "Success",
							"content": {
								"text/plain": {},
								"application/json": {
									"example": {"status": "ok"}
								}
							}
						}
					}
				}
			}
		}
	}`

	result := convertOpenAPIToBlueprint(t, input)
	// Should include at least the application/json content type
	if !strings.Contains(result, "application/json") {
		t.Error("Missing content type in output")
	}
}

func TestConvert_WithPutPatchOperations(t *testing.T) {
	input := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/items/{id}": {
				"put": {
					"summary": "Replace item",
					"parameters": [
						{
							"name": "id",
							"in": "path",
							"required": true,
							"schema": {"type": "string"}
						}
					],
					"responses": {
						"200": {"description": "Replaced"}
					}
				},
				"patch": {
					"summary": "Update item",
					"responses": {
						"200": {"description": "Updated"}
					}
				}
			}
		}
	}`

	result := convertOpenAPIToBlueprint(t, input)
	if !strings.Contains(result, "[PUT]") {
		t.Error("Missing PUT method")
	}
	if !strings.Contains(result, "[PATCH]") {
		t.Error("Missing PATCH method")
	}
}

func TestConvert_WithResponseNoContentType(t *testing.T) {
	input := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/status": {
				"get": {
					"summary": "Get status",
					"responses": {
						"204": {
							"description": "No Content"
						},
						"404": {
							"description": "Not Found"
						}
					}
				}
			}
		}
	}`

	result := convertOpenAPIToBlueprint(t, input)
	if !strings.Contains(result, "Response 204") {
		t.Error("Missing 204 response")
	}
	if !strings.Contains(result, "Response 404") {
		t.Error("Missing 404 response")
	}
}

func TestConvert_WithQueryParameters(t *testing.T) {
	input := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/search": {
				"get": {
					"summary": "Search",
					"parameters": [
						{
							"name": "query",
							"in": "query",
							"description": "Search query",
							"required": true,
							"schema": {"type": "string"}
						},
						{
							"name": "limit",
							"in": "query",
							"description": "Result limit",
							"required": false,
							"schema": {"type": "integer"}
						}
					],
					"responses": {
						"200": {"description": "Success"}
					}
				}
			}
		}
	}`

	result := convertOpenAPIToBlueprint(t, input)
	if !strings.Contains(result, "+ Parameters") {
		t.Error("Missing Parameters section")
	}
	if !strings.Contains(result, "query") {
		t.Error("Missing query parameter")
	}
	if !strings.Contains(result, "limit") {
		t.Error("Missing limit parameter")
	}
	if !strings.Contains(result, "required") {
		t.Error("Missing required indicator")
	}
	if !strings.Contains(result, "optional") {
		t.Error("Missing optional indicator")
	}
}

func TestConvert_TitleCaseEdgeCases(t *testing.T) {
	input := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/test": {
				"get": {
					"summary": "a",
					"responses": {
						"200": {"description": "Success"}
					}
				}
			}
		}
	}`

	result := convertOpenAPIToBlueprint(t, input)
	// Single letter should be handled
	if !strings.Contains(result, "a") || !strings.Contains(result, "A") {
		t.Error("Single letter summary not handled correctly")
	}
}

func TestConvert_NoServers(t *testing.T) {
	input := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/test": {
				"get": {
					"summary": "Test",
					"responses": {
						"200": {"description": "Success"}
					}
				}
			}
		}
	}`

	result := convertOpenAPIToBlueprint(t, input)
	// Should work without servers
	if !strings.Contains(result, "# Test API") {
		t.Error("Missing API title")
	}
}

func TestConvert_NoDescription(t *testing.T) {
	input := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/test": {
				"get": {
					"responses": {
						"200": {"description": "Success"}
					}
				}
			}
		}
	}`

	result := convertOpenAPIToBlueprint(t, input)
	// Should work without operation summary/description
	if !strings.Contains(result, "# Test API") {
		t.Error("Missing API title")
	}
}
