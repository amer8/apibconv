package converter

import (
	"bytes"
	"strings"
	"testing"
)

// This file compares different approaches to buffer management and string building

// BenchmarkBufferStrategies compares different buffer management approaches
func BenchmarkBufferStrategies(b *testing.B) {
	spec := OpenAPI{
		Info: Info{
			Title:       "Test API",
			Description: "A test API for benchmarking",
			Version:     "1.0.0",
		},
		Servers: []Server{{URL: "https://api.example.com"}},
		Paths: map[string]PathItem{
			"/users":      {},
			"/posts":      {},
			"/comments":   {},
			"/tags":       {},
			"/categories": {},
		},
	}

	b.Run("SyncPoolReuseBuffer", func(b *testing.B) {
		// Current approach: sync.Pool buffer reuse
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := getBuffer()
			writeAPIBlueprint(buf, &spec)
			_ = buf.Bytes()
			putBuffer(buf)
		}
	})

	b.Run("NewBufferEachTime", func(b *testing.B) {
		// Alternative: Create new buffer each time
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := new(bytes.Buffer)
			writeAPIBlueprint(buf, &spec)
			_ = buf.Bytes()
		}
	})

	b.Run("PreallocatedBuffer", func(b *testing.B) {
		// Alternative: Pre-allocated buffer with estimated size
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := bytes.NewBuffer(make([]byte, 0, 2048))
			writeAPIBlueprint(buf, &spec)
			_ = buf.Bytes()
		}
	})

	b.Run("BytesBuilderNoPool", func(b *testing.B) {
		// Alternative: strings.Builder (similar to bytes.Buffer but for strings)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var builder strings.Builder
			builder.Grow(2048)
			// Would need a version of writeAPIBlueprint for strings.Builder
			builder.WriteString("FORMAT: 1A\n\n")
			builder.WriteString("# ")
			builder.WriteString(spec.Title())
			_ = builder.String()
		}
	})

	b.Run("DirectByteSlice", func(b *testing.B) {
		// Alternative: Direct byte slice manipulation
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := make([]byte, 0, 2048)
			result = append(result, "FORMAT: 1A\n\n"...)
			result = append(result, "# "...)
			result = append(result, spec.Title()...)
			_ = result
		}
	})
}

// BenchmarkStringConcatenation compares string building approaches
func BenchmarkStringConcatenation(b *testing.B) {
	title := "My API"
	description := "This is a test API for benchmarking different approaches"
	host := "https://api.example.com"

	b.Run("BytesBuffer_WriteString", func(b *testing.B) {
		// Current approach: bytes.Buffer with WriteString
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := getBuffer()
			buf.WriteString("# ")
			buf.WriteString(title)
			buf.WriteString("\n\n")
			buf.WriteString(description)
			buf.WriteString("\n\n")
			buf.WriteString("HOST: ")
			buf.WriteString(host)
			buf.WriteString("\n\n")
			_ = buf.Bytes()
			putBuffer(buf)
		}
	})

	b.Run("StringsBuilder", func(b *testing.B) {
		// Alternative: strings.Builder
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var builder strings.Builder
			builder.Grow(256)
			builder.WriteString("# ")
			builder.WriteString(title)
			builder.WriteString("\n\n")
			builder.WriteString(description)
			builder.WriteString("\n\n")
			builder.WriteString("HOST: ")
			builder.WriteString(host)
			builder.WriteString("\n\n")
			_ = builder.String()
		}
	})

	b.Run("StringConcatenation", func(b *testing.B) {
		// Naive approach: string concatenation
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := "# " + title + "\n\n" +
				description + "\n\n" +
				"HOST: " + host + "\n\n"
			_ = result
		}
	})

	b.Run("ByteSliceAppend", func(b *testing.B) {
		// Alternative: byte slice with append
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := make([]byte, 0, 256)
			result = append(result, "# "...)
			result = append(result, title...)
			result = append(result, "\n\n"...)
			result = append(result, description...)
			result = append(result, "\n\n"...)
			result = append(result, "HOST: "...)
			result = append(result, host...)
			result = append(result, "\n\n"...)
			_ = result
		}
	})

	b.Run("FmtSprintf", func(b *testing.B) {
		// Worst approach: fmt.Sprintf
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := "# " + title + "\n\n" +
				description + "\n\n" +
				"HOST: " + host + "\n\n"
			_ = result
		}
	})
}

// BenchmarkBufferPoolVsNoPool directly compares pool vs no pool
func BenchmarkBufferPoolVsNoPool(b *testing.B) {
	data := []byte("This is test data that we'll write to buffers repeatedly")

	b.Run("WithSyncPool", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := getBuffer()
				buf.Write(data)
				buf.WriteString("\nMore data")
				_ = buf.Bytes()
				putBuffer(buf)
			}
		})
	})

	b.Run("WithoutSyncPool", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := new(bytes.Buffer)
				buf.Write(data)
				buf.WriteString("\nMore data")
				_ = buf.Bytes()
			}
		})
	})

	b.Run("WithSyncPool_Sequential", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := getBuffer()
			buf.Write(data)
			buf.WriteString("\nMore data")
			_ = buf.Bytes()
			putBuffer(buf)
		}
	})

	b.Run("WithoutSyncPool_Sequential", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := new(bytes.Buffer)
			buf.Write(data)
			buf.WriteString("\nMore data")
			_ = buf.Bytes()
		}
	})
}

// BenchmarkMemoryEfficiency compares memory usage patterns
func BenchmarkMemoryEfficiency(b *testing.B) {
	b.Run("Pool_SmallWrites", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := getBuffer()
			for j := 0; j < 10; j++ {
				buf.WriteString("small ")
			}
			_ = buf.Bytes()
			putBuffer(buf)
		}
	})

	b.Run("NoPool_SmallWrites", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := new(bytes.Buffer)
			for j := 0; j < 10; j++ {
				buf.WriteString("small ")
			}
			_ = buf.Bytes()
		}
	})

	b.Run("Pool_LargeWrites", func(b *testing.B) {
		largeData := strings.Repeat("This is a larger piece of data. ", 100)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := getBuffer()
			buf.WriteString(largeData)
			_ = buf.Bytes()
			putBuffer(buf)
		}
	})

	b.Run("NoPool_LargeWrites", func(b *testing.B) {
		largeData := strings.Repeat("This is a larger piece of data. ", 100)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := new(bytes.Buffer)
			buf.WriteString(largeData)
			_ = buf.Bytes()
		}
	})
}
