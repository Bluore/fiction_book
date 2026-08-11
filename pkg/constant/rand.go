package constant

import (
	"math/rand/v2"
	"time"
)

var (
	RandQuick = rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0))
)
