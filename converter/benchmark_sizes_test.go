package converter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"strings"
	"testing"
)

// generateOpenAPISpec generates an OpenAPI spec with the specified number of paths and operations.
// This is useful for testing performance with different spec sizes.
func generateOpenAPISpec(numPaths, operationsPerPath int) *OpenAPI {
	spec := &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       "Benchmark API",
			Description: "API for benchmarking with varying sizes",
			Version:     "1.0.0",
		},
		Servers: []Server{{URL: "https://api.example.com"}},
		Paths:   make(map[string]PathItem, numPaths),
	}

	for i := 0; i < numPaths; i++ {
		path := fmt.Sprintf("/resource%d/{id}", i)
		pathItem := PathItem{}

		for j := 0; j < operationsPerPath; j++ {
			op := &Operation{
				Summary:     fmt.Sprintf("Operation %d for resource %d", j, i),
				Description: fmt.Sprintf("Detailed description for operation %d on resource %d. This operation performs various tasks.", j, i),
				Parameters: []Parameter{
					{
						Name:        "id",
						In:          "path",
						Required:    true,
						Description: "Resource identifier",
						Schema:      &Schema{Type: "string"},
					},
					{
						Name:        "limit",
						In:          "query",
						Required:    false,
						Description: "Maximum number of results",
						Schema:      &Schema{Type: "integer"},
					},
				},
				Responses: map[string]Response{
					"200": {
						Description: "Successful response",
						Content: map[string]MediaType{
							"application/json": {
								Schema: &Schema{
									Type: "object",
									Properties: map[string]*Schema{
										"id":   {Type: "string"},
										"name": {Type: "string"},
										"data": {Type: "object"},
									},
								},
								Example: map[string]any{
									"id":   "123",
									"name": "Example",
									"data": map[string]string{"key": "value"},
								},
							},
						},
					},
					"404": {
						Description: "Not found",
					},
				},
			}

			switch j % 5 {
			case 0:
				pathItem.Get = op
			case 1:
				pathItem.Post = op
				pathItem.Post.RequestBody = &RequestBody{
					Required: true,
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{
								Type: "object",
								Properties: map[string]*Schema{
									"name": {Type: "string"},
									"data": {Type: "object"},
								},
							},
						},
					},
				}
			case 2:
				pathItem.Put = op
			case 3:
				pathItem.Delete = op
			case 4:
				pathItem.Patch = op
			}
		}

		spec.Paths[path] = pathItem
	}

	return spec
}

// SpecSize represents a specification size category
type SpecSize struct {
	Name              string
	NumPaths          int
	OperationsPerPath int
}

var specSizes = []SpecSize{
	{"Tiny", 1, 1},
	{"Small", 5, 2},
	{"Medium", 20, 3},
	{"Large", 50, 4},
	{"XLarge", 100, 5},
}

// BenchmarkConvertOpenAPIToAPIBlueprint_Sizes benchmarks conversion with different spec sizes
func BenchmarkConvertOpenAPIToAPIBlueprint_Sizes(b *testing.B) {
	for _, size := range specSizes {
		spec := generateOpenAPISpec(size.NumPaths, size.OperationsPerPath)

		// Marshal to JSON for conversion
		jsonData, err := json.Marshal(spec)
		if err != nil {
			b.Fatalf("Failed to marshal spec: %v", err)
		}

		b.Run(size.Name, func(b *testing.B) {
			b.SetBytes(int64(len(jsonData)))
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				reader := bytes.NewReader(jsonData)
				if err := Convert(reader, io.Discard); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkFormatOpenAPI_Sizes benchmarks the Format function with different spec sizes
func BenchmarkFormatOpenAPI_Sizes(b *testing.B) {
	for _, size := range specSizes {
		spec := generateOpenAPISpec(size.NumPaths, size.OperationsPerPath)

		b.Run(size.Name, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := Format(spec)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkParseOpenAPI_Sizes benchmarks parsing with different spec sizes
func BenchmarkParseOpenAPI_Sizes(b *testing.B) {
	for _, size := range specSizes {
		spec := generateOpenAPISpec(size.NumPaths, size.OperationsPerPath)
		jsonData, _ := json.Marshal(spec)

		b.Run(size.Name, func(b *testing.B) {
			b.SetBytes(int64(len(jsonData)))
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := Parse(jsonData)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkMarshalYAML_Sizes benchmarks YAML marshaling with different spec sizes
func BenchmarkMarshalYAML_Sizes(b *testing.B) {
	for _, size := range specSizes {
		spec := generateOpenAPISpec(size.NumPaths, size.OperationsPerPath)

		b.Run(size.Name, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := MarshalYAML(spec)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkValidateOpenAPI_Sizes benchmarks validation with different spec sizes
func BenchmarkValidateOpenAPI_Sizes(b *testing.B) {
	for _, size := range specSizes {
		spec := generateOpenAPISpec(size.NumPaths, size.OperationsPerPath)

		b.Run(size.Name, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = ValidateOpenAPI(spec)
			}
		})
	}
}

// BenchmarkAPIBlueprintParse_Sizes benchmarks API Blueprint parsing with different sizes
func BenchmarkAPIBlueprintParse_Sizes(b *testing.B) {
	for _, size := range specSizes {
		// Generate API Blueprint content
		var builder strings.Builder
		builder.WriteString("FORMAT: 1A\n\n# Benchmark API\n\nAPI for benchmarking\n\nHOST: https://api.example.com\n\n")

		for i := 0; i < size.NumPaths; i++ {
			builder.WriteString(fmt.Sprintf("## Group Resource%d\n\n", i))
			builder.WriteString(fmt.Sprintf("## Resource%d [/resource%d/{id}]\n\n", i, i))

			for j := 0; j < size.OperationsPerPath; j++ {
				methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
				method := methods[j%5]

				builder.WriteString(fmt.Sprintf("### %s Resource%d [%s]\n\n", method, i, method))
				builder.WriteString("+ Parameters\n")
				builder.WriteString("    + id (required, string) - Resource identifier\n\n")
				builder.WriteString("+ Response 200 (application/json)\n\n")
				builder.WriteString("        {\"id\": \"123\", \"name\": \"Example\"}\n\n")
			}
		}

		content := builder.String()

		b.Run(size.Name, func(b *testing.B) {
			b.SetBytes(int64(len(content)))
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				reader := strings.NewReader(content)
				_, err := ParseAPIBlueprintReader(reader)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// MemoryStats captures memory statistics for profiling
type MemoryStats struct {
	HeapAlloc   uint64
	HeapObjects uint64
	TotalAlloc  uint64
	NumGC       uint32
}

// captureMemoryStats captures current memory statistics
func captureMemoryStats() MemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return MemoryStats{
		HeapAlloc:   m.HeapAlloc,
		HeapObjects: m.HeapObjects,
		TotalAlloc:  m.TotalAlloc,
		NumGC:       m.NumGC,
	}
}

// BenchmarkMemoryProfile_Convert profiles memory usage during conversion
func BenchmarkMemoryProfile_Convert(b *testing.B) {
	for _, size := range specSizes {
		spec := generateOpenAPISpec(size.NumPaths, size.OperationsPerPath)
		jsonData, _ := json.Marshal(spec)

		b.Run(size.Name, func(b *testing.B) {
			// Force GC before starting
			runtime.GC()
			before := captureMemoryStats()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				reader := bytes.NewReader(jsonData)
				if err := Convert(reader, io.Discard); err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			runtime.GC()
			after := captureMemoryStats()

			// Report custom metrics
			b.ReportMetric(float64(after.HeapAlloc-before.HeapAlloc)/float64(b.N), "heap-bytes/op")
			b.ReportMetric(float64(after.HeapObjects-before.HeapObjects)/float64(b.N), "heap-objects/op")
		})
	}
}

// BenchmarkMemoryProfile_Parse profiles memory usage during parsing
func BenchmarkMemoryProfile_Parse(b *testing.B) {
	for _, size := range specSizes {
		spec := generateOpenAPISpec(size.NumPaths, size.OperationsPerPath)
		jsonData, _ := json.Marshal(spec)

		b.Run(size.Name, func(b *testing.B) {
			runtime.GC()
			before := captureMemoryStats()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := Parse(jsonData)
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			runtime.GC()
			after := captureMemoryStats()

			b.ReportMetric(float64(after.HeapAlloc-before.HeapAlloc)/float64(b.N), "heap-bytes/op")
			b.ReportMetric(float64(after.HeapObjects-before.HeapObjects)/float64(b.N), "heap-objects/op")
		})
	}
}

// BenchmarkMemoryProfile_YAML profiles memory usage during YAML conversion
func BenchmarkMemoryProfile_YAML(b *testing.B) {
	for _, size := range specSizes {
		spec := generateOpenAPISpec(size.NumPaths, size.OperationsPerPath)

		b.Run(size.Name, func(b *testing.B) {
			runtime.GC()
			before := captureMemoryStats()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := MarshalYAML(spec)
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			runtime.GC()
			after := captureMemoryStats()

			b.ReportMetric(float64(after.HeapAlloc-before.HeapAlloc)/float64(b.N), "heap-bytes/op")
		})
	}
}

// BenchmarkThroughput_Convert measures throughput in MB/s for conversion
func BenchmarkThroughput_Convert(b *testing.B) {
	// Use a large spec for throughput testing
	spec := generateOpenAPISpec(100, 5)
	jsonData, _ := json.Marshal(spec)
	dataSize := int64(len(jsonData))

	b.SetBytes(dataSize)
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(jsonData)
		if err := Convert(reader, io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkThroughput_Parse measures throughput in MB/s for parsing
func BenchmarkThroughput_Parse(b *testing.B) {
	spec := generateOpenAPISpec(100, 5)
	jsonData, _ := json.Marshal(spec)
	dataSize := int64(len(jsonData))

	b.SetBytes(dataSize)
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := Parse(jsonData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConcurrent_Convert tests concurrent conversion performance
func BenchmarkConcurrent_Convert(b *testing.B) {
	spec := generateOpenAPISpec(20, 3)
	jsonData, _ := json.Marshal(spec)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			reader := bytes.NewReader(jsonData)
			if err := Convert(reader, io.Discard); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkConcurrent_Parse tests concurrent parsing performance
func BenchmarkConcurrent_Parse(b *testing.B) {
	spec := generateOpenAPISpec(20, 3)
	jsonData, _ := json.Marshal(spec)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := Parse(jsonData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkConcurrent_BufferPool tests buffer pool under concurrent load
func BenchmarkConcurrent_BufferPool(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := getBuffer()
			buf.WriteString("test data for buffer pool benchmarking")
			putBuffer(buf)
		}
	})
}
