package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestQueuedSubmitUsesCapturedAssignedDifficultyAfterStateChanges(t *testing.T) {
	mc := benchmarkMinerConnForSubmit(NewPoolMetrics())
	conn := &recordConn{}
	mc.conn = conn
	mc.lockDifficulty = true
	mc.cfg.ShareCheckDuplicate = false

	job := benchmarkSubmitJobForTest(t)
	stratumJobID := "notify-job-1"
	mc.activeJobs = map[string]*Job{stratumJobID: job}
	mc.lastJob = job
	mc.lastJobID = stratumJobID
	mc.jobDifficulty[stratumJobID] = 1
	mc.jobScriptTime = map[string]int64{stratumJobID: job.ScriptTime}

	ntimeHex := fmt.Sprintf("%08x", uint32(job.Template.CurTime))
	req := &StratumRequest{
		ID:     1,
		Method: "mining.submit",
		Params: []any{mc.currentWorker(), stratumJobID, "00000000", ntimeHex, "00000000"},
	}
	task, ok := mc.prepareSubmissionTask(req, time.Unix(job.Template.CurTime, 0))
	if !ok {
		t.Fatalf("prepareSubmissionTask failed")
	}
	if task.assignedDifficulty != 1 {
		t.Fatalf("assigned difficulty got %g want 1", task.assignedDifficulty)
	}

	// Simulate async queue delay: a later notify/difficulty change evicts or
	// changes live connection state before this already-parsed task is processed.
	delete(mc.jobDifficulty, stratumJobID)
	atomicStoreFloat64(&mc.difficulty, 1024)
	mc.shareTarget.Store(targetFromDifficulty(1024))

	mc.processShare(task, shareContext{
		hashHex:   strings.Repeat("0", 64),
		shareDiff: 1,
		isBlock:   false,
	})

	out := conn.String()
	if !strings.Contains(out, `"result":true`) {
		t.Fatalf("expected share accepted using captured difficulty, got: %q", out)
	}
	if strings.Contains(out, "low difficulty share") {
		t.Fatalf("share was evaluated against changed live difficulty: %q", out)
	}
}
