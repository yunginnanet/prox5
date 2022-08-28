package pools

import (
	"strings"
	"sync"
)

var CopABuffer = &sync.Pool{New: func() interface{} { return &strings.Builder{} }}

func DiscardBuffer(buf *strings.Builder) {
	buf.Reset()
	CopABuffer.Put(buf)
}
