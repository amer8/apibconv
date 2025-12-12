package converter

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

const benchmarkSpec = `{
	"openapi": "3.0.0",
	"info": {
		"title": "Benchmark API",
		"description": "API for benchmarking",
		"version": "1.0.0"
	},
	"servers": [
		{
			"url": "https://api.example.com",
			"description": "Production server"
		}
	],
	"paths": {
		"/users": {
			"get": {
				"summary": "List users",
				"description": "Get a list of all users",
				"parameters": [
					{
						"name": "limit",
						"in": "query",
						"description": "Maximum number of users to return",
						"required": false,
						"schema": {
							"type": "integer"
						}
					}
				],
				"responses": {
					"200": {
						"description": "Successful response",
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
			},
			"post": {
				"summary": "Create user",
				"description": "Create a new user",
				"requestBody": {
					"required": true,
					"content": {
						"application/json": {
							"example": {
								"name": "Charlie",
								"email": "charlie@example.com"
							}
						}
					}
				},
				"responses": {
					"201": {
						"description": "User created",
						"content": {
							"application/json": {
								"example": {
									"id": 3,
									"name": "Charlie"
								}
							}
						}
					}
				}
			}
		},
		"/users/{id}": {
			"get": {
				"summary": "Get user",
				"description": "Get a specific user by ID",
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
						"description": "User found",
						"content": {
							"application/json": {
								"example": {
									"id": 1,
									"name": "Alice"
								}
							}
						}
					},
					"404": {
						"description": "User not found"
					}
				}
			},
			"put": {
				"summary": "Update user",
				"description": "Update an existing user",
				"parameters": [
					{
						"name": "id",
						"in": "path",
						"required": true,
						"schema": {
							"type": "integer"
						}
					}
				],
				"requestBody": {
					"required": true,
					"content": {
						"application/json": {
							"example": {
								"name": "Alice Updated"
							}
						}
					}
				},
				"responses": {
					"200": {
						"description": "User updated"
					}
				}
			},
			"delete": {
				"summary": "Delete user",
				"description": "Delete a user",
				"parameters": [
					{
						"name": "id",
						"in": "path",
						"required": true,
						"schema": {
							"type": "integer"
						}
					}
				],
				"responses": {
					"204": {
						"description": "User deleted"
					}
				}
			}
		}
	}
}`

// BenchmarkConvert benchmarks the conversion process
func BenchmarkConvert(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(benchmarkSpec)
		writer := io.Discard
		data, _ := io.ReadAll(reader)
		s, err := Parse(data, FormatOpenAPI)
		if err != nil {
			b.Fatal(err)
		}
		if openapiSpec, ok := s.AsOpenAPI(); ok {
			            if err := openapiSpec.WriteBlueprint(writer); err != nil {
			                b.Fatal(err)
			            }
			        } else {
			            b.Fatal("spec is not an OpenAPI spec")
			        }	}
}

// BenchmarkConvertWithBuffer benchmarks with actual buffer writes
func BenchmarkConvertWithBuffer(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(benchmarkSpec)
		var writer bytes.Buffer
		data, _ := io.ReadAll(reader)
		s, err := Parse(data, FormatOpenAPI)
		if err != nil {
			b.Fatal(err)
		}
		if openapiSpec, ok := s.AsOpenAPI(); ok {
			if err := openapiSpec.WriteBlueprint(&writer); err != nil {
				b.Fatal(err)
			}
		} else {
			b.Fatal("spec is not an OpenAPI spec")
		}
	}
}

// BenchmarkBufferPool benchmarks the buffer pool operations
func BenchmarkBufferPool(b *testing.B) {
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := getBuffer()
			buf.WriteString("test data")
			putBuffer(buf)
		}
	})
}

// BenchmarkWriteAPIBlueprint benchmarks just the writing portion
// This benchmark focuses on zero-allocation buffer operations
func BenchmarkWriteAPIBlueprint(b *testing.B) {
	// Use a minimal spec for benchmarking
	spec := OpenAPI{
		Info: Info{
			Title:       "Benchmark API",
			Description: "API for benchmarking",
			Version:     "1.0.0",
		},
		Servers: []Server{{URL: "https://api.example.com"}},
		Paths:   make(map[string]PathItem),
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := getBuffer()
		writeAPIBlueprint(buf, &spec)
		putBuffer(buf)
	}
}
