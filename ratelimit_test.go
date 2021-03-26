package ratelimit

import (
	"testing"
	"time"
)

func TestRateLimit_Singleton(t *testing.T) {
	limiterBaseAddr := GetRateLimiter()
	if limiterBaseAddr == nil {
		t.Fatal("rate limiter pointer is nil")
	}

	for i := 0; i < 5; i++ {
		limiter := GetRateLimiter()
		if limiter != limiterBaseAddr {
			t.Errorf("limiter pointers not equal - singleton violated (instance %d)", i)
		}
	}
}

type TestSchedule struct {
	runCnt int
}

func (t *TestSchedule) handle(closure *func()) {
	t.runCnt++
	(*closure)()
}

func TestRateLimit_Limit(t *testing.T) {
	limiter := GetRateLimiter()

	// verify nil closure doesn't crash
	limiter.Limit(nil, &TestSchedule{})

	// verify nil closure doesn't crash
	emptyClosure := func() {}
	limiter.Limit(&emptyClosure, nil)

	// verify that closure runs on first Limit call
	hasRun := false
	closure := func() {
		hasRun = true
	}
	testSchedule := TestSchedule{}
	limiter.Limit(&closure, &testSchedule)
	if !hasRun {
		t.Error("closure did not run")
	}

	// verify that the test schedule run count is accurate through multiple
	// read/write operations
	expectedRunCnt := 5
	for i := 1; i < expectedRunCnt; i++ {
		limiter.Limit(&closure, &testSchedule)
	}

	scheduleRaw, found := limiter.store.Load(&closure)
	if !found {
		t.Fatal("kv pair not found")
	}
	scheduleOut := scheduleRaw.(*TestSchedule)
	if scheduleOut.runCnt != expectedRunCnt {
		t.Errorf("expected run count %d, got %d", expectedRunCnt, scheduleOut.runCnt)
	}
}

func TestLimit_Modulo(t *testing.T) {
	limiter := GetRateLimiter()
	limit := ModuloLimit{Mod: 3}

	runCnt := 0
	closure := func() {
		runCnt++
	}

	// verify Modulo runs the closure on the first call
	limiter.Limit(&closure, &limit)
	if runCnt != 1 {
		t.Fatalf("expected run count 1, got %d", runCnt)
	}

	// we expect 2 additional calls here
	for i := 1; i <= 2*limit.Mod; i++ {
		limiter.Limit(&closure, &limit)
	}

	if runCnt != 3 {
		t.Errorf("bad run count. expected 3, got %d", runCnt)
	}
}

func TestLimit_NPerTime(t *testing.T) {
	limiter := GetRateLimiter()
	limit := NPerTimeLimit{
		N:         5,
		TimeLimit: 1 * time.Second,
	}

	runCnt := 0
	closure := func() {
		runCnt++
	}

	// verify it runs N times in the first time interval and no more
	for i := 0; i < 2*limit.N; i++ {
		limiter.Limit(&closure, &limit)
	}
	if runCnt != limit.N {
		t.Fatalf("bad run count. expected %d, got %d", limit.N, runCnt)
	}

	// sleep long enough for the first window to close
	time.Sleep(2 * time.Second)
	runCnt = 0
	for i := 0; i < 2*limit.N; i++ {
		limiter.Limit(&closure, &limit)
	}
	if runCnt != limit.N {
		t.Fatalf("bad run count in second window. expected %d, got %d", limit.N, runCnt)
	}
}

func TestLimit_OneOfN(t *testing.T) {
	limiter := GetRateLimiter()
	limit := OneOfNLimit{
		N: 3,
	}

	hasRun := false
	closure := func() {
		hasRun = true
	}

	// could be flakey, but it should run at least once
	for i := 0; i < 3*limit.N; i++ {
		limiter.Limit(&closure, &limit)
	}
	if !hasRun {
		t.Errorf("closure did not run when expected to")
	}
}

func TestLimit_Quota(t *testing.T) {
	limiter := GetRateLimiter()
	limit := QuotaLimit{
		Quota: 3,
	}

	runCnt := 0
	closure := func() {
		runCnt++
	}

	for i := 0; i < 2*limit.Quota; i++ {
		limiter.Limit(&closure, &limit)
	}
	if runCnt != limit.Quota {
		t.Errorf("bad run count. expected %d, got %d", limit.Quota, runCnt)
	}
}
