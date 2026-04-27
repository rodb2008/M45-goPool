package main

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

func TestBuildCurrentJobCoinbaseDetail_UsesExactSinglePayoutPath(t *testing.T) {
	_, poolScript := generateTestWallet(t)
	workerName, workerAddr, workerScript := generateTestWorker(t)

	job := &Job{
		Template: GetBlockTemplateResult{
			Height: 120000,
		},
		JobID:                   "job-1",
		Extranonce2Size:         4,
		TemplateExtraNonce2Size: 4,
		CoinbaseValue:           50 * 1e8,
		PayoutScript:            poolScript,
		ScriptTime:              1000,
	}
	mc := &MinerConn{
		cfg:         Config{PoolFeePercent: 0},
		extranonce1: []byte{0xaa, 0xbb, 0xcc, 0xdd},
		jobScriptTime: map[string]int64{
			"job-1": 2000,
		},
	}
	mc.updateWorker(workerName)
	mc.setWorkerWallet(workerName, workerAddr, workerScript)
	coinb1, coinb2, err := buildCoinbaseParts(
		job.Template.Height,
		mc.extranonce1,
		job.Extranonce2Size,
		job.TemplateExtraNonce2Size,
		workerScript,
		job.CoinbaseValue,
		job.WitnessCommitment,
		job.Template.CoinbaseAux.Flags,
		job.CoinbaseMsg,
		2000,
	)
	if err != nil {
		t.Fatalf("buildCoinbaseParts: %v", err)
	}
	mc.jobNotifyCoinbase = map[string]notifiedCoinbaseParts{
		job.JobID: {coinb1: coinb1, coinb2: coinb2},
	}

	detail := mc.buildCurrentJobCoinbaseDetail(job)
	if detail == nil || detail.Coinbase == "" {
		t.Fatalf("buildCurrentJobCoinbaseDetail failed")
	}

	gotRaw, err := hex.DecodeString(detail.Coinbase)
	if err != nil {
		t.Fatalf("decode detail coinbase: %v", err)
	}
	expectedRaw, _, err := serializeCoinbaseTx(
		job.Template.Height,
		mc.extranonce1,
		[]byte{0, 0, 0, 0},
		job.TemplateExtraNonce2Size,
		workerScript,
		job.CoinbaseValue,
		job.WitnessCommitment,
		job.Template.CoinbaseAux.Flags,
		job.CoinbaseMsg,
		2000,
	)
	if err != nil {
		t.Fatalf("serialize expected coinbase: %v", err)
	}
	if !bytes.Equal(gotRaw, expectedRaw) {
		t.Fatalf("coinbase mismatch between display path and miner build path")
	}
}

func TestBuildCurrentJobCoinbaseDetail_UsesLatestStratumNotifyID(t *testing.T) {
	mc, _ := minerConnForNotifyTest(t)
	job := benchmarkSubmitJobForTest(t)

	mc.sendNotifyFor(job, true)

	mc.jobMu.Lock()
	if _, ok := mc.jobNotifyCoinbase[job.JobID]; ok {
		t.Fatalf("expected notify coinbase to be keyed by emitted stratum job id, not template job id")
	}
	parts, ok := mc.jobNotifyCoinbase[mc.lastJobID]
	mc.jobMu.Unlock()
	if !ok {
		t.Fatalf("missing notify coinbase for last stratum job id")
	}

	detail := mc.buildCurrentJobCoinbaseDetail(job)
	if detail == nil || detail.Coinbase == "" {
		t.Fatalf("buildCurrentJobCoinbaseDetail failed for latest stratum job id")
	}
	expected := parts.coinb1 + hex.EncodeToString(mc.extranonce1) + strings.Repeat("00", job.Extranonce2Size) + parts.coinb2
	if detail.Coinbase != expected {
		t.Fatalf("coinbase detail mismatch\n got: %s\nwant: %s", detail.Coinbase, expected)
	}
}
