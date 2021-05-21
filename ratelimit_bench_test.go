package ratelimit

import (
	"testing"
)

func BenchmarkRateLimit_Limit(b *testing.B) {
	limiter := GetRateLimiter()
	closure := func() {}
	limit := ModuloLimit{Mod: 1}

	for i := 0; i < b.N; i++ {
		limiter.Limit(&closure, &limit)
	}
}

// Test to see if full map is slower than empty - shouldn't be due to map hash?
func BenchmarkRateLimit_MultipleEntries(b *testing.B) {
	limiter := GetRateLimiter()

	// populate the map with some entries
	for i := 0; i < 1000; i++ {
		closure := func() {}
		limit := ModuloLimit{Mod: 1}

		limiter.Limit(&closure, &limit)
	}

	testClosure := func() {}
	testLimit := ModuloLimit{Mod: 1}
	for i := 0; i < b.N; i++ {
		limiter.Limit(&testClosure, &testLimit)
	}
}

func BenchmarkRateLimit_Control(b *testing.B) {
	closure := func() {}

	for i := 0; i < b.N; i++ {
		closure()
	}
}

func BenchmarkRateLimit_GetLimiter(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// TODO Maybe this unnamed assignment prevents the optimizer
		// from optimizing away the result, and by extension the call.
		_ = GetRateLimiter()
	}
}
