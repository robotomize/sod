package util

import (
	"crypto/sha256"
	"strconv"
)

func HashVector(vec []float64) [32]byte {
	buffer := GetBytesBuffer()
	defer PutBytesBuffer(buffer)
	defer buffer.Reset()
	for i := range vec {
		buffer.WriteString(strconv.FormatFloat(vec[i], 'g', 16, 64))
	}
	return sha256.Sum256(buffer.Bytes())
}
