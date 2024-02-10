package mr

import (
	"sync/atomic"
	"time"
)

type SafeTime struct {
	tick atomic.Int64
}

func (st *SafeTime) Set(t time.Time) {
	st.tick.Store(t.Unix())
}

func (st *SafeTime) Get() time.Time {
	tick := st.tick.Load()
	return time.Unix(tick, 0)
}
