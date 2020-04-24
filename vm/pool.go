package vm

import "sync"

var TSPool = &sync.Pool{New: GetBatch}

func GetBatch() interface{} {
	return &TimeSeries{}
}