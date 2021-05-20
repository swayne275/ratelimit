package ratelimit

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

/*
Future improvement ideas:
- hot-swappable limiters?
- constructors for Limiter structs (error checking)?
*/

var (
	once     sync.Once
	instance RateLimiter
)

type RateLimiter struct {
	store  sync.Map
	logger *log.Logger
}

type Limiter interface {
	handle(*func())
}

// ModuloCfg is used for a "run this 1 out of N times" form of rate limiting
type ModuloLimit struct {
	Mod int

	// internal tracking of how many instances since the last run
	cnt int
}

func (m *ModuloLimit) handle(closure *func()) {
	m.cnt++
	if m.cnt > m.Mod {
		m.cnt -= m.Mod
	}

	if m.cnt == 1 {
		(*closure)()
	}
}

type NPerTimeLimit struct {
	// Run the closure the first N times per the `TimeLimit`
	N int
	// How much time must pass before resetting the run count
	TimeLimit time.Duration

	timeSegmentStart time.Time
	cntInSegment     int
}

func (n *NPerTimeLimit) handle(closure *func()) {
	now := time.Now()

	if n.timeSegmentStart.Add(n.TimeLimit).Before(now) {
		// TODO what's the better use case, this or snapping it to the most
		// recent contiguous segment?
		n.timeSegmentStart = now
		n.cntInSegment = 0
	}

	if n.cntInSegment < n.N {
		(*closure)()
		n.cntInSegment++
	}
}

// Run, on average, 1 out of N times called
type OneOfNLimit struct {
	N int
}

func (o *OneOfNLimit) handle(closure *func()) {
	if 1+rand.Intn(o.N) == o.N {
		(*closure)()
	}
}

type QuotaLimit struct {
	Quota int

	// track how many instances of `Quota` have been used
	numConsumed int
}

func (q *QuotaLimit) handle(closure *func()) {
	if q.numConsumed < q.Quota {
		(*closure)()
		q.numConsumed++
	}
}

func GetRateLimiter() *RateLimiter {
	once.Do(func() {
		instance = RateLimiter{
			logger: log.Default(),
		}
	})

	return &instance
}

// Limit takes a function closure to call as per the given `schedule`
// This avoids the reflection package and expanding a list of arguments to pass
// into limit
func (r *RateLimiter) Limit(closure *func(), limit Limiter) {
	if closure == nil {
		r.logger.Println("nil closure pointer given")
		return
	}
	if limit == nil {
		r.logger.Println("nil limit given")
		return
	}

	scheduleRaw, _ := r.store.LoadOrStore(closure, limit)
	scheduleToUse := scheduleRaw.(Limiter)
	scheduleToUse.handle(closure)

	// save the schedule back with updated parameters
	r.store.Store(closure, scheduleToUse)
}
