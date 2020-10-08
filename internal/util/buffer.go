package util

import (
	"bytes"
	"sync"
)

var bytesBuffer = sync.Pool{
	New: func() interface{} { return &bytes.Buffer{} },
}

func GetBytesBuffer() (p *bytes.Buffer) {
	ifc := bytesBuffer.Get()
	if ifc != nil {
		p = ifc.(*bytes.Buffer)
	}
	return
}

func PutBytesBuffer(p *bytes.Buffer) {
	bytesBuffer.Put(p)
}
