package main

import "testing"

func TestStatusServerSetJobManagerHandlesNil(t *testing.T) {
	var nilServer *StatusServer
	nilServer.SetJobManager(nil)

	s := &StatusServer{}
	jm := &JobManager{}
	s.SetJobManager(jm)
	if s.jobMgr != jm {
		t.Fatalf("job manager was not attached")
	}
	if jm.onNewBlock == nil {
		t.Fatalf("expected new-block callback to be installed")
	}

	s.SetJobManager(nil)
	if s.jobMgr != nil {
		t.Fatalf("job manager was not cleared")
	}
}
