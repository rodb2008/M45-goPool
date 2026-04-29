package main

import (
	"sync/atomic"
	"testing"
)

func TestJobManagerNextJobID_Base58CounterStartsAtZero(t *testing.T) {
	jm := &JobManager{}

	if got := jm.nextJobID(); got != "1" {
		t.Fatalf("unexpected first job id: got %q, want %q", got, "1")
	}
	if got := jm.nextJobID(); got != "2" {
		t.Fatalf("unexpected second job id: got %q, want %q", got, "2")
	}
	if got := jm.nextJobID(); got != "3" {
		t.Fatalf("unexpected third job id: got %q, want %q", got, "3")
	}
}

func TestJobManagerNextJobID_RollsAfterSixBase58Digits(t *testing.T) {
	jm := &JobManager{}
	atomic.StoreUint64(&jm.jobIDCounter, jobIDRolloverModulo-1)

	lastSixDigitID := jm.nextJobID()
	if len(lastSixDigitID) > 6 {
		t.Fatalf("expected rollover boundary id to fit in six chars, got %q", lastSixDigitID)
	}

	if got := jm.nextJobID(); got != "1" {
		t.Fatalf("expected job id to roll over to %q, got %q", "1", got)
	}
}
