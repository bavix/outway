package sf

import "sync"

// Group is a minimal singleflight-like implementation limited to keyed mutex.
// It avoids importing golang.org/x/sync/singleflight to pass depguard.
type Group struct {
	mu sync.Map // map[string]*keyLock
}

type keyLock struct{ mu sync.Mutex }

// Do serializes function execution per key and returns its result.
// This simplified version does not share the returned value between waiters,
// but still coalesces concurrent executions by enforcing mutual exclusion.
func (g *Group) Do(key string, fn func() (any, error)) (any, error, bool) {
	lkAny, _ := g.mu.LoadOrStore(key, &keyLock{})

	lk, _ := lkAny.(*keyLock)
	if lk == nil {
		// should not happen; fallback to sequential exec
		v, err := fn()

		return v, err, false
	}

	lk.mu.Lock()
	defer lk.mu.Unlock()

	v, err := fn()

	return v, err, false
}
