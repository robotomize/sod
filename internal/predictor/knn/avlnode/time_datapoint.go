package avlnode

import (
	"time"

	"github.com/go-sod/sod/internal/predictor"
	"github.com/go-sod/sod/pkg/container/avltree"
)

type TimeNode struct {
	K time.Time
	V predictor.DataPoint
}

func (i TimeNode) Key() interface{} {
	return i.K
}

func (i TimeNode) Value() interface{} {
	return i.V
}

func (i TimeNode) Subtraction(item avltree.Item) int {
	if i.K.Equal(item.(TimeNode).K) {
		return 0
	}

	if i.K.Before(item.(TimeNode).K) {
		return -1
	}
	return 1
}
