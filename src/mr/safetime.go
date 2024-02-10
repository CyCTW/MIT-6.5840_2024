package mr

import (
	"sync"
	"time"
)

type SafeTime struct {
	mu    sync.Mutex
	value time.Time
}

func (st *SafeTime) Set(t time.Time) {
	st.mu.Lock()
	st.value = t
	st.mu.Unlock()
}

func (st *SafeTime) Get() time.Time {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.value
}
