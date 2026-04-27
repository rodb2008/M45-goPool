package main

import (
	"strings"
	"testing"
)

func TestHandleGetTransactions_ReturnsTxidsForJobID(t *testing.T) {
	conn := &writeRecorderConn{}
	job := &Job{
		JobID: "job1",
		Transactions: []GBTTransaction{
			{Txid: "aa"},
			{Txid: "bb"},
		},
	}

	mc := &MinerConn{
		id:         "get-txs",
		conn:       conn,
		activeJobs: map[string]*Job{"job1": job},
		lastJob:    job,
	}

	req := &StratumRequest{ID: 1, Method: "mining.get_transactions", Params: []any{"job1"}}
	mc.handleGetTransactions(req)

	out := conn.String()
	if !strings.Contains(out, "\"result\":[\"aa\",\"bb\"]") {
		t.Fatalf("expected txids in result, got: %q", out)
	}
}

func TestHandleGetTransactions_EmptyParamsUsesLastJob(t *testing.T) {
	conn := &writeRecorderConn{}
	job := &Job{
		JobID: "jobLast",
		Transactions: []GBTTransaction{
			{Txid: "cc"},
		},
	}

	mc := &MinerConn{
		id:         "get-txs-last",
		conn:       conn,
		activeJobs: map[string]*Job{"jobLast": job},
		lastJob:    job,
	}

	req := &StratumRequest{ID: 1, Method: "mining.get_transactions", Params: nil}
	mc.handleGetTransactions(req)

	out := conn.String()
	if !strings.Contains(out, "\"result\":[\"cc\"]") {
		t.Fatalf("expected txids from last job, got: %q", out)
	}
}

func TestHandleGetTransactions_ReturnsTxidsForStratumNotifyJobID(t *testing.T) {
	mc, notifyConn := minerConnForNotifyTest(t)
	job := benchmarkSubmitJobForTest(t)
	job.Transactions = []GBTTransaction{
		{Txid: "dd"},
		{Txid: "ee"},
	}

	mc.sendNotifyFor(job, true)
	ids := notifyJobIDsFromOutput(t, notifyConn.String())
	if len(ids) != 1 {
		t.Fatalf("expected one notify id, got %#v", ids)
	}

	respConn := &writeRecorderConn{}
	mc.conn = respConn
	req := &StratumRequest{ID: 1, Method: "mining.get_transactions", Params: []any{ids[0]}}
	mc.handleGetTransactions(req)

	out := respConn.String()
	if !strings.Contains(out, "\"result\":[\"dd\",\"ee\"]") {
		t.Fatalf("expected txids for emitted stratum job id, got: %q", out)
	}
}
