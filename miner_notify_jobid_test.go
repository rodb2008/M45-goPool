package main

import (
	"strings"
	"testing"
)

func notifyJobIDsFromOutput(t *testing.T, out string) []string {
	t.Helper()
	var ids []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var msg StratumMessage
		if err := fastJSONUnmarshal([]byte(line), &msg); err != nil {
			t.Fatalf("decode stratum message: %v; line=%q", err, line)
		}
		if msg.Method != "mining.notify" {
			continue
		}
		if len(msg.Params) == 0 {
			t.Fatalf("notify without params: %q", line)
		}
		id, ok := msg.Params[0].(string)
		if !ok || id == "" {
			t.Fatalf("notify job id is not a non-empty string: %#v", msg.Params[0])
		}
		ids = append(ids, id)
	}
	return ids
}

func TestSendNotifyForUsesUniqueStratumJobIDsForRepeatedNotify(t *testing.T) {
	mc := benchmarkMinerConnForSubmit(NewPoolMetrics())
	conn := &recordConn{}
	mc.conn = conn
	mc.lockDifficulty = true
	mc.maxRecentJobs = 10
	mc.activeJobs = make(map[string]*Job, mc.maxRecentJobs)
	mc.jobOrder = make([]string, 0, mc.maxRecentJobs)
	mc.jobDifficulty = make(map[string]float64, mc.maxRecentJobs)
	mc.jobScriptTime = make(map[string]int64, mc.maxRecentJobs)
	mc.jobNotifyCoinbase = make(map[string]notifiedCoinbaseParts, mc.maxRecentJobs)

	job := benchmarkSubmitJobForTest(t)
	job.ScriptTime = job.Template.CurTime

	mc.sendNotifyFor(job, true)
	mc.sendNotifyFor(job, true)

	ids := notifyJobIDsFromOutput(t, conn.String())
	if len(ids) != 2 {
		t.Fatalf("expected two notify job ids, got %d: %#v", len(ids), ids)
	}
	if ids[0] == ids[1] {
		t.Fatalf("expected repeated notifies to use distinct job ids, got %q", ids[0])
	}
	if ids[0] == job.JobID || ids[1] == job.JobID {
		t.Fatalf("expected emitted Stratum job ids to be per-notify ids, base=%q ids=%#v", job.JobID, ids)
	}

	firstJob, _, _, _, _, firstScriptTime, firstOK := mc.jobForIDWithLast(ids[0])
	secondJob, _, _, _, _, secondScriptTime, secondOK := mc.jobForIDWithLast(ids[1])
	if !firstOK || !secondOK || firstJob != job || secondJob != job {
		t.Fatalf("notify ids did not resolve to the underlying job")
	}
	if firstScriptTime == 0 || secondScriptTime == 0 || firstScriptTime == secondScriptTime {
		t.Fatalf("expected immutable per-notify script times, got first=%d second=%d", firstScriptTime, secondScriptTime)
	}
}
