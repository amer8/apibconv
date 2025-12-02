package converter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"
)

// generateAsyncAPISpec generates an AsyncAPI 2.6 spec with the specified number of channels.
func generateAsyncAPISpec(numChannels int) *AsyncAPI {
	spec := &AsyncAPI{
		AsyncAPI: "2.6.0",
		Info: Info{
			Title:       "Benchmark AsyncAPI",
			Description: "AsyncAPI for benchmarking with varying sizes",
			Version:     "1.0.0",
		},
		Servers: map[string]AsyncAPIServer{
			"production": {
				URL:      "mqtt://api.example.com:1883",
				Protocol: "mqtt",
			},
		},
		Channels: make(map[string]Channel, numChannels),
	}

	for i := 0; i < numChannels; i++ {
		channelName := fmt.Sprintf("events/resource%d", i)
		
		spec.Channels[channelName] = Channel{
			Description: fmt.Sprintf("Channel for resource %d", i),
			Subscribe: &AsyncAPIOperation{
				Summary: fmt.Sprintf("Subscribe to resource %d", i),
				Message: &Message{
					Name:        fmt.Sprintf("Resource%dEvent", i),
					ContentType: "application/json",
					Payload: &Schema{
						Type: "object",
						Properties: map[string]*Schema{
							"id":        {Type: "string"},
							"timestamp": {Type: "string"},
							"data":      {Type: "object"},
						},
					},
				},
			},
			Publish: &AsyncAPIOperation{
				Summary: fmt.Sprintf("Publish to resource %d", i),
				Message: &Message{
					Name:        fmt.Sprintf("Resource%dCommand", i),
					ContentType: "application/json",
					Payload: &Schema{
						Type: "object",
						Properties: map[string]*Schema{
							"command": {Type: "string"},
							"payload": {Type: "object"},
						},
					},
				},
			},
		}
	}

	return spec
}

// BenchmarkParseAsyncAPI_Sizes benchmarks parsing with different spec sizes
func BenchmarkParseAsyncAPI_Sizes(b *testing.B) {
	for _, size := range specSizes {
		spec := generateAsyncAPISpec(size.NumPaths) // reusing NumPaths as NumChannels
		jsonData, err := json.Marshal(spec)
		if err != nil {
			b.Fatalf("Failed to marshal spec: %v", err)
		}

		b.Run(size.Name, func(b *testing.B) {
			b.SetBytes(int64(len(jsonData)))
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := ParseAsyncAPI(jsonData)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkAsyncAPIToAPIBlueprint_Sizes benchmarks conversion with different spec sizes
func BenchmarkAsyncAPIToAPIBlueprint_Sizes(b *testing.B) {
	for _, size := range specSizes {
		spec := generateAsyncAPISpec(size.NumPaths)

		b.Run(size.Name, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = AsyncAPIToAPIBlueprint(spec)
			}
		})
	}
}

// BenchmarkConcurrent_ParseAsyncAPI tests concurrent parsing performance
func BenchmarkConcurrent_ParseAsyncAPI(b *testing.B) {
	spec := generateAsyncAPISpec(20) // Medium size
	jsonData, _ := json.Marshal(spec)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := ParseAsyncAPI(jsonData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkConcurrent_ConvertAsyncAPI tests concurrent conversion performance
func BenchmarkConcurrent_ConvertAsyncAPI(b *testing.B) {
	spec := generateAsyncAPISpec(20) // Medium size
	jsonData, _ := json.Marshal(spec)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			reader := bytes.NewReader(jsonData)
			if err := ConvertAsyncAPIToAPIBlueprint(reader, io.Discard); err != nil {
				b.Fatal(err)
			}
		}
	})
}
