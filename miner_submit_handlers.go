package main

import "time"

func (mc *MinerConn) handleSubmit(req *StratumRequest) {
	// Expect params like:
	// [worker_name, job_id, extranonce2, ntime, nonce]
	now := time.Now()

	task, ok := mc.prepareSubmissionTask(req, now)
	if !ok {
		return
	}
	if mc.cfg.SubmitProcessInline {
		mc.processSubmissionTask(task)
		return
	}
	ensureSubmissionWorkerPool()
	submissionWorkers.submit(task)
}
