package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAuthorizePasswordMatchesVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pass     string
		expected string
		ok       bool
	}{
		{pass: "x", expected: "x", ok: true},
		{pass: "x,d=1024", expected: "x", ok: true},
		{pass: "d=1024,x", expected: "x", ok: true},
		{pass: "pass=x,d=1024", expected: "x", ok: true},
		{pass: "password=x;d=1024", expected: "x", ok: true},
		{pass: "d=1024", expected: "x", ok: false},
		{pass: "x", expected: "secret", ok: false},
	}

	for _, tc := range tests {
		got := authorizePasswordMatches(tc.pass, tc.expected)
		if got != tc.ok {
			t.Fatalf("authorizePasswordMatches(%q, %q) = %v, want %v", tc.pass, tc.expected, got, tc.ok)
		}
	}
}

func TestParsePasswordDifficultyHintVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pass string
		diff float64
		ok   bool
	}{
		{pass: "x,d=1024", diff: 1024, ok: true},
		{pass: "diff:64", diff: 64, ok: true},
		{pass: "difficulty=2.5", diff: 2.5, ok: true},
		{pass: "sd=0x400", diff: 1024, ok: true},
		{pass: "x", diff: 0, ok: false},
		{pass: "d=0", diff: 0, ok: false},
		{pass: "d=abc", diff: 0, ok: false},
	}

	for _, tc := range tests {
		diff, ok := parsePasswordDifficultyHint(tc.pass)
		if ok != tc.ok {
			t.Fatalf("parsePasswordDifficultyHint(%q) ok=%v, want %v", tc.pass, ok, tc.ok)
		}
		if ok && diff != tc.diff {
			t.Fatalf("parsePasswordDifficultyHint(%q) diff=%v, want %v", tc.pass, diff, tc.diff)
		}
	}
}

func TestParseSuggestedDifficultyVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
		diff  float64
		ok    bool
	}{
		{name: "string", value: "256", diff: 256, ok: true},
		{name: "hex_string", value: "0x400", diff: 1024, ok: true},
		{name: "int", value: 16, diff: 16, ok: true},
		{name: "json_number", value: jsonNumber("128"), diff: 128, ok: true},
		{name: "invalid", value: "abc", diff: 0, ok: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseSuggestedDifficulty(tc.value)
			if ok != tc.ok {
				t.Fatalf("parseSuggestedDifficulty(%v) ok=%v, want %v", tc.value, ok, tc.ok)
			}
			if ok && got != tc.diff {
				t.Fatalf("parseSuggestedDifficulty(%v) diff=%v, want %v", tc.value, got, tc.diff)
			}
		})
	}
}

func TestParseWorkerDifficultyHintVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		worker        string
		clean         string
		diff          float64
		ok            bool
		wantUnchanged bool
	}{
		{name: "plus_number", worker: "wallet.worker+1024", clean: "wallet.worker", diff: 1024, ok: true},
		{name: "plus_keyed_hex", worker: "wallet.worker+d=0x400", clean: "wallet.worker", diff: 1024, ok: true},
		{name: "hash_keyed_float", worker: "wallet.worker#difficulty=2.5", clean: "wallet.worker", diff: 2.5, ok: true},
		{name: "comma_keyed", worker: "wallet.worker,diff:64", clean: "wallet.worker", diff: 64, ok: true},
		{name: "space_separated", worker: "wallet.worker + 128", clean: "wallet.worker", diff: 128, ok: true},
		{name: "non_diff_suffix", worker: "wallet.worker+garage", clean: "wallet.worker+garage", diff: 0, ok: false, wantUnchanged: true},
		{name: "zero_ignored", worker: "wallet.worker+0", clean: "wallet.worker+0", diff: 0, ok: false, wantUnchanged: true},
		{name: "no_delim", worker: "wallet.worker", clean: "wallet.worker", diff: 0, ok: false, wantUnchanged: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clean, diff, ok := parseWorkerDifficultyHint(tc.worker)
			if ok != tc.ok {
				t.Fatalf("parseWorkerDifficultyHint(%q) ok=%v, want %v", tc.worker, ok, tc.ok)
			}
			if ok && diff != tc.diff {
				t.Fatalf("parseWorkerDifficultyHint(%q) diff=%v, want %v", tc.worker, diff, tc.diff)
			}
			if ok && clean != tc.clean {
				t.Fatalf("parseWorkerDifficultyHint(%q) clean=%q, want %q", tc.worker, clean, tc.clean)
			}
			if !ok && tc.wantUnchanged && clean != tc.worker {
				t.Fatalf("parseWorkerDifficultyHint(%q) clean=%q, want unchanged", tc.worker, clean)
			}
		})
	}
}

func TestHandleConfigureSupportsVariantShapes(t *testing.T) {
	conn := &writeRecorderConn{}
	mc := &MinerConn{
		id:       "configure-variants",
		conn:     conn,
		poolMask: 0x1fffe000,
	}

	req := &StratumRequest{
		ID:     1,
		Method: "mining.configure",
		Params: []any{
			"version_rolling, suggest_difficulty",
			map[string]any{
				"version_rolling_mask":          "1fffe000",
				"version_rolling_min_bit_count": "2",
			},
		},
	}
	mc.handleConfigure(req)

	if !mc.versionRoll {
		t.Fatalf("expected version rolling to be enabled")
	}
	if mc.versionMask == 0 {
		t.Fatalf("expected negotiated version mask to be non-zero")
	}
	out := conn.String()
	if !strings.Contains(out, "\"version-rolling\":true") {
		t.Fatalf("expected configure response to enable version-rolling, got: %q", out)
	}
	if !strings.Contains(out, "\"suggest_difficulty\":true") {
		t.Fatalf("expected configure response to acknowledge suggest_difficulty, got: %q", out)
	}
}

func TestHandleConfigureSupportsBitaxeVersionRollingShape(t *testing.T) {
	conn := &writeRecorderConn{}
	mc := &MinerConn{
		id:       "configure-bitaxe",
		conn:     conn,
		poolMask: 0x1fffe000,
		cfg:      Config{ShareAllowDegradedVersionBits: true},
	}

	req := &StratumRequest{
		ID:     10,
		Method: "mining.configure",
		Params: []any{
			[]any{"version-rolling"},
			map[string]any{"version-rolling.mask": "ffffffff"},
		},
	}
	mc.handleConfigure(req)

	if !mc.versionRoll {
		t.Fatalf("expected version rolling to be enabled")
	}
	if got, want := mc.versionMask, uint32(0x1fffe000); got != want {
		t.Fatalf("versionMask=%08x want=%08x", got, want)
	}
	out := conn.String()
	if !strings.Contains(out, "\"version-rolling.mask\":\"1fffe000\"") {
		t.Fatalf("expected configure response to include negotiated mask, got: %q", out)
	}
	if !strings.Contains(out, "\"version-rolling.min-bit-count\":1") {
		t.Fatalf("expected configure response to include min bit count, got: %q", out)
	}
}

func TestHandleConfigureSubscribeExtranonceSendsSetExtranonce(t *testing.T) {
	conn := &writeRecorderConn{}
	mc := &MinerConn{
		id:             "configure-extranonce",
		conn:           conn,
		extranonce1Hex: "abcdef01",
		subscribed:     true,
		cfg:            Config{Extranonce2Size: 4},
	}

	req := &StratumRequest{
		ID:     1,
		Method: "mining.configure",
		Params: []any{"subscribe-extranonce"},
	}
	mc.handleConfigure(req)

	if !mc.extranonceSubscribed {
		t.Fatalf("expected extranonceSubscribed to be enabled")
	}
	out := conn.String()
	if !strings.Contains(out, "\"subscribe-extranonce\":true") {
		t.Fatalf("expected configure response to acknowledge subscribe-extranonce, got: %q", out)
	}
	if !strings.Contains(out, "\"method\":\"mining.set_extranonce\"") {
		t.Fatalf("expected set_extranonce to be sent after configure, got: %q", out)
	}
}

func TestHandleConfigureResponsePrecedesVersionMaskAndInitialNotify(t *testing.T) {
	workerName, workerWallet, workerScript := generateTestWorker(t)
	job := benchmarkSubmitJobForTest(t)
	job.JobID = "compat-order-job"
	job.ScriptTime = job.Template.CurTime
	jm := &JobManager{curJob: job}

	conn := &writeRecorderConn{}
	mc := &MinerConn{
		id:             "configure-order",
		conn:           conn,
		cfg:            Config{Extranonce2Size: 4, DefaultDifficulty: 1},
		poolMask:       0x1fffe000,
		versionMask:    0x1fffe000,
		extranonce1:    []byte{0x01, 0x02, 0x03, 0x04},
		extranonce1Hex: "01020304",
		authorized:     true,
		subscribed:     true,
		listenerOn:     true,
		jobMgr:         jm,
		lockDifficulty: true,
		stats: MinerStats{
			Worker:       workerName,
			WorkerSHA256: workerNameHash(workerName),
		},
		activeJobs:        make(map[string]*Job, 4),
		jobOrder:          make([]string, 0, 4),
		jobDifficulty:     make(map[string]float64, 4),
		jobScriptTime:     make(map[string]int64, 4),
		jobNotifyCoinbase: make(map[string]notifiedCoinbaseParts, 4),
		maxRecentJobs:     4,
	}
	mc.setWorkerWallet(workerName, workerWallet, workerScript)
	mc.scheduleInitialWork()

	req := &StratumRequest{
		ID:     11,
		Method: "mining.configure",
		Params: []any{
			[]any{"version-rolling"},
			map[string]any{"version-rolling.mask": "ffffffff"},
		},
	}
	mc.handleConfigure(req)

	lines := nonEmptyLines(conn.String())
	if len(lines) < 3 {
		t.Fatalf("expected configure response, set_version_mask, and notify; got %d lines: %#v", len(lines), lines)
	}
	var first StratumResponse
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("decode first line as response: %v; line=%q", err, lines[0])
	}
	if first.ID != float64(11) || first.Error != nil {
		t.Fatalf("first line should be configure response for id 11, got %#v", first)
	}
	if !strings.Contains(lines[1], "\"method\":\"mining.set_version_mask\"") {
		t.Fatalf("second line should be set_version_mask after configure response, got: %q", lines[1])
	}
	notifyIdx := -1
	for i, line := range lines[2:] {
		if strings.Contains(line, "\"method\":\"mining.notify\"") {
			notifyIdx = i + 2
			break
		}
	}
	if notifyIdx < 0 {
		t.Fatalf("expected initial notify after configure response, got lines: %#v", lines)
	}
}

func TestLateConfigureMinimumDifficultyAppliesCleanJob(t *testing.T) {
	workerName, workerWallet, workerScript := generateTestWorker(t)
	job := benchmarkSubmitJobForTest(t)
	job.JobID = "minimum-difficulty-job"
	job.ScriptTime = job.Template.CurTime
	jm := &JobManager{curJob: job}

	conn := &writeRecorderConn{}
	mc := &MinerConn{
		id:             "configure-min-diff",
		conn:           conn,
		cfg:            Config{Extranonce2Size: 4, MinDifficulty: 1},
		extranonce1:    []byte{0x01, 0x02, 0x03, 0x04},
		extranonce1Hex: "01020304",
		authorized:     true,
		subscribed:     true,
		listenerOn:     true,
		jobMgr:         jm,
		lockDifficulty: true,
		stats: MinerStats{
			Worker:       workerName,
			WorkerSHA256: workerNameHash(workerName),
		},
		activeJobs:        make(map[string]*Job, 4),
		jobOrder:          make([]string, 0, 4),
		jobDifficulty:     make(map[string]float64, 4),
		jobScriptTime:     make(map[string]int64, 4),
		jobNotifyCoinbase: make(map[string]notifiedCoinbaseParts, 4),
		maxRecentJobs:     4,
		initialWorkSent:   true,
	}
	mc.setWorkerWallet(workerName, workerWallet, workerScript)
	atomicStoreFloat64(&mc.difficulty, 1)
	mc.shareTarget.Store(targetFromDifficulty(1))

	mc.handleConfigure(&StratumRequest{
		ID:     13,
		Method: "mining.configure",
		Params: []any{
			[]any{"minimum-difficulty"},
			map[string]any{"minimum-difficulty.value": 128},
		},
	})

	lines := nonEmptyLines(conn.String())
	if len(lines) < 3 {
		t.Fatalf("expected configure response, set_difficulty, and clean notify; got lines: %#v", lines)
	}
	var first StratumResponse
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("decode first line as response: %v; line=%q", err, lines[0])
	}
	if first.ID != float64(13) || first.Error != nil {
		t.Fatalf("first line should be configure response for id 13, got %#v", first)
	}
	out := conn.String()
	if !strings.Contains(out, "\"method\":\"mining.set_difficulty\"") || !strings.Contains(out, "\"params\":[128]") {
		t.Fatalf("expected integer set_difficulty for minimum floor, got: %q", out)
	}
	if !strings.Contains(out, "\"method\":\"mining.notify\"") {
		t.Fatalf("expected clean notify after applying minimum difficulty, got: %q", out)
	}
	if got := mc.currentDifficulty(); got != 128 {
		t.Fatalf("currentDifficulty=%v want 128", got)
	}
}

func TestPreSubscribeConfigureQueuesVersionMaskUntilSetup(t *testing.T) {
	conn := &writeRecorderConn{}
	mc := &MinerConn{
		id:       "configure-before-subscribe",
		conn:     conn,
		poolMask: 0x1fffe000,
		cfg:      Config{ShareAllowDegradedVersionBits: true},
	}

	mc.handleConfigure(&StratumRequest{
		ID:     12,
		Method: "mining.configure",
		Params: []any{
			[]any{"version-rolling"},
			map[string]any{"version-rolling.mask": "ffffffff"},
		},
	})

	out := conn.String()
	if strings.Contains(out, "\"method\":\"mining.set_version_mask\"") {
		t.Fatalf("set_version_mask should be deferred until subscribed, got: %q", out)
	}
	if !mc.pendingVersionMask {
		t.Fatalf("expected version mask notification to be queued")
	}

	mc.subscribed = true
	mc.sendPendingStratumSetup()
	if !strings.Contains(conn.String(), "\"method\":\"mining.set_version_mask\"") {
		t.Fatalf("expected queued set_version_mask after subscription, got: %q", conn.String())
	}
}

func TestSuggestDifficultyNullIDQueuesDifficultyWithoutResponse(t *testing.T) {
	conn := &writeRecorderConn{}
	mc := &MinerConn{
		id:   "suggest-null-id",
		conn: conn,
		cfg:  Config{MinDifficulty: 1},
	}

	mc.suggestDifficulty(&StratumRequest{
		ID:     nil,
		Method: "mining.suggest_difficulty",
		Params: []any{128},
	})

	if out := strings.TrimSpace(conn.String()); out != "" {
		t.Fatalf("expected no response for id:null suggest_difficulty, got: %q", out)
	}
	if !mc.pendingDifficulty {
		t.Fatalf("expected difficulty notification to be queued")
	}
	if got := mc.currentDifficulty(); got != 128 {
		t.Fatalf("current difficulty = %v, want 128", got)
	}

	mc.subscribed = true
	mc.sendPendingStratumSetup()
	if !strings.Contains(conn.String(), "\"method\":\"mining.set_difficulty\"") {
		t.Fatalf("expected queued set_difficulty after subscription, got: %q", conn.String())
	}
	if !strings.Contains(conn.String(), "\"params\":[128]") {
		t.Fatalf("expected integer difficulty on wire, got: %q", conn.String())
	}
}

func TestSubscribeResponseAdvertisesSetExtranonce(t *testing.T) {
	conn := &writeRecorderConn{}
	mc := &MinerConn{
		id:   "subscribe-advertise-extranonce",
		conn: conn,
		cfg:  Config{CKPoolEmulate: false},
	}

	mc.writeSubscribeResponse(1, "00", 4, "sid")

	out := conn.String()
	if !strings.Contains(out, "\"mining.set_extranonce\"") {
		t.Fatalf("expected subscribe response to advertise set_extranonce, got: %q", out)
	}
}

func TestHandleSubscribeDoesNotSendUnsolicitedExtranonceNotification(t *testing.T) {
	conn := &writeRecorderConn{}
	mc := &MinerConn{
		id:             "subscribe-no-unsolicited-extranonce",
		conn:           conn,
		extranonce1Hex: "01020304",
		cfg:            Config{Extranonce2Size: 4, CKPoolEmulate: false},
		jobMgr:         &JobManager{},
	}

	mc.handleSubscribe(&StratumRequest{
		ID:     1,
		Method: "mining.subscribe",
		Params: []any{"ESP32Compat/1.0"},
	})

	out := conn.String()
	if strings.Contains(out, "\"method\":\"mining.set_extranonce\"") {
		t.Fatalf("subscribe should not send unsolicited set_extranonce notification, got: %q", out)
	}
	if got := len(nonEmptyLines(out)); got != 1 {
		t.Fatalf("expected only subscribe response before explicit opt-in, got %d lines: %q", got, out)
	}
}

func TestHandleSubscribeWithoutJobManagerDoesNotPanic(t *testing.T) {
	conn := &writeRecorderConn{}
	mc := &MinerConn{
		id:             "subscribe-no-job-manager",
		conn:           conn,
		extranonce1Hex: "01020304",
		cfg:            Config{Extranonce2Size: 4, CKPoolEmulate: false},
	}

	mc.handleSubscribe(&StratumRequest{
		ID:     2,
		Method: "mining.subscribe",
		Params: []any{"compat-test/1.0"},
	})

	out := conn.String()
	if !strings.Contains(out, "\"id\":2") {
		t.Fatalf("expected subscribe response, got: %q", out)
	}
	if strings.Contains(out, "\"method\":\"mining.notify\"") {
		t.Fatalf("subscribe without job manager should not send notify, got: %q", out)
	}
}

func TestDefaultConfigUsesESP32SafeStratumDefaults(t *testing.T) {
	cfg := defaultConfig()
	if !cfg.CKPoolEmulate {
		t.Fatalf("expected ckpool emulate default to keep subscribe response minimal")
	}
	if cfg.Extranonce2Size != 4 {
		t.Fatalf("expected ESP32-friendly extranonce2_size default 4, got %d", cfg.Extranonce2Size)
	}
	if !cfg.ShareAllowDegradedVersionBits {
		t.Fatalf("expected degraded version bits to be allowed by default for version-rolling compatibility")
	}
}

func nonEmptyLines(out string) []string {
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
