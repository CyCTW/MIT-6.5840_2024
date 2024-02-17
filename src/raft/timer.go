package raft

import "time"

// ResettableTicker struct encapsulates the ticker logic
type ResettableTicker struct {
	duration time.Duration
	ticker   *time.Ticker
	reset    chan bool
	stop     chan bool
}

// NewResettableTicker creates a new ResettableTicker
func NewResettableTicker(duration time.Duration) *ResettableTicker {
	return &ResettableTicker{
		duration: duration,
		reset:    make(chan bool),
		stop:     make(chan bool),
	}
}

// Start begins the ticker and waits for it to be reset or stopped. It performs the action
// continuously until stopped or reset.
func (t *ResettableTicker) Start(action func()) {

}

// Reset stops the current ticker and starts a new one, effectively resetting the countdown.
func (t *ResettableTicker) Reset() {
	t.reset <- true
}

// Stop stops the ticker and ends the goroutine, preventing any further actions from being executed.
func (t *ResettableTicker) Stop() {
	t.stop <- true
}
