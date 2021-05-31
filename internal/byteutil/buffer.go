package byteutil

import (
	"bytes"
	"sync"
)

var bytesBuffer = sync.Pool{
	New: func() interface{} { return &bytes.Buffer{} },
}

func GetBytesBuf() (p *bytes.Buffer) {
	ifc := bytesBuffer.Get()
	if ifc != nil {
		p = ifc.(*bytes.Buffer)
	}
	return
}

func PutBytesBuf(p *bytes.Buffer) {
	bytesBuffer.Put(p)
}
