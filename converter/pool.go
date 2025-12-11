package converter

import (
	"bytes"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// getBuffer gets a buffer from the pool
func getBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// putBuffer returns a buffer to the pool
func putBuffer(buf *bytes.Buffer) {
	bufferPool.Put(buf)
}
