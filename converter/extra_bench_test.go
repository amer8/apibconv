package converter

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

// Helper to generate API Blueprint content (copied/adapted from benchmark_sizes_test.go)
func generateAPIBlueprintContent(size SpecSize) string {
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
	return builder.String()
}

// BenchmarkAPIBlueprintToOpenAPI_Sizes benchmarks full conversion from APIB to OpenAPI JSON
func BenchmarkAPIBlueprintToOpenAPI_Sizes(b *testing.B) {
	for _, size := range specSizes {
		content := generateAPIBlueprintContent(size)

		b.Run(size.Name, func(b *testing.B) {
			b.SetBytes(int64(len(content)))
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				reader := strings.NewReader(content)
				spec, err := ParseBlueprintReader(reader)
				if err != nil {
					b.Fatal(err)
				}
				if err := json.NewEncoder(io.Discard).Encode(spec); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkAPIBlueprintToAsyncAPI_Sizes benchmarks full conversion from APIB to AsyncAPI JSON
func BenchmarkAPIBlueprintToAsyncAPI_Sizes(b *testing.B) {
	for _, size := range specSizes {
		content := generateAPIBlueprintContent(size)

		b.Run(size.Name, func(b *testing.B) {
			b.SetBytes(int64(len(content)))
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				reader := strings.NewReader(content)
				spec, err := ParseBlueprintReader(reader)
				if err != nil {
					b.Fatal(err)
				}
				asyncSpec, err := spec.ToAsyncAPI("ws")
				if err != nil {
					b.Fatal(err)
				}
				if err := json.NewEncoder(io.Discard).Encode(asyncSpec); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkAPIBlueprintToAsyncAPIV3_Sizes benchmarks full conversion from APIB to AsyncAPI v3 JSON
func BenchmarkAPIBlueprintToAsyncAPIV3_Sizes(b *testing.B) {
	for _, size := range specSizes {
		content := generateAPIBlueprintContent(size)

		b.Run(size.Name, func(b *testing.B) {
			b.SetBytes(int64(len(content)))
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				reader := strings.NewReader(content)
				spec, err := ParseBlueprintReader(reader)
				if err != nil {
					b.Fatal(err)
				}
				asyncSpec, err := spec.ToAsyncAPIV3("ws")
				if err != nil {
					b.Fatal(err)
				}
				if err := json.NewEncoder(io.Discard).Encode(asyncSpec); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
