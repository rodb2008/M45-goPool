package main

import (
	"testing"
	"time"
)

func TestStratumRateLimit_DoubledThreshold(t *testing.T) {
	now := time.Now()
	mc := &MinerConn{
		cfg:         Config{StratumMessagesPerMinute: 10},
		connectedAt: now.Add(-10 * time.Minute),
	}

	for i := range 20 {
		if mc.stratumMsgRateLimitExceeded(now, "mining.authorize") {
			t.Fatalf("unexpected limit hit at message %d", i+1)
		}
	}
	if !mc.stratumMsgRateLimitExceeded(now, "mining.authorize") {
		t.Fatalf("expected limit to trigger on message 21 with doubled threshold")
	}
}

func TestStratumRateLimit_EarlySubmitCountsHalf(t *testing.T) {
	now := time.Now()
	mc := &MinerConn{
		cfg:         Config{StratumMessagesPerMinute: 10},
		connectedAt: now.Add(-1 * time.Minute),
	}

	for i := range 40 {
		if mc.stratumMsgRateLimitExceeded(now, "mining.submit") {
			t.Fatalf("unexpected limit hit at early submit %d", i+1)
		}
	}
	if !mc.stratumMsgRateLimitExceeded(now, "mining.submit") {
		t.Fatalf("expected limit to trigger on early submit 41 (half weight)")
	}
}

func TestStratumRateLimit_SubmitAfterWarmupCountsFull(t *testing.T) {
	now := time.Now()
	mc := &MinerConn{
		cfg:         Config{StratumMessagesPerMinute: 10},
		connectedAt: now.Add(-(earlySubmitHalfWeightWindow + time.Minute)),
	}

	for i := range 20 {
		if mc.stratumMsgRateLimitExceeded(now, "mining.submit") {
			t.Fatalf("unexpected limit hit at post-warmup submit %d", i+1)
		}
	}
	if !mc.stratumMsgRateLimitExceeded(now, "mining.submit") {
		t.Fatalf("expected limit to trigger on post-warmup submit 21")
	}
}

func TestStratumRateLimit_WindowReset(t *testing.T) {
	now := time.Now()
	mc := &MinerConn{
		cfg:         Config{StratumMessagesPerMinute: 10},
		connectedAt: now.Add(-10 * time.Minute),
	}

	for i := range 20 {
		if mc.stratumMsgRateLimitExceeded(now, "mining.authorize") {
			t.Fatalf("unexpected limit hit before reset at message %d", i+1)
		}
	}
	if mc.stratumMsgRateLimitExceeded(now.Add(61*time.Second), "mining.authorize") {
		t.Fatalf("expected rate limit window to reset after one minute")
	}
}

func TestWorkerForRateLimitBan_EmptyWhenCurrentEmpty(t *testing.T) {
	mc := &MinerConn{}
	got := mc.workerForRateLimitBan()
	if got != "" {
		t.Fatalf("expected empty worker when none is current, got %q", got)
	}
}

func TestWorkerForRateLimitBan_PrefersCurrentWorker(t *testing.T) {
	mc := &MinerConn{}
	mc.updateWorker("persisted.worker")
	got := mc.workerForRateLimitBan()
	if got != "persisted.worker" {
		t.Fatalf("expected current worker to be preferred, got %q", got)
	}
}
