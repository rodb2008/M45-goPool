package main

import (
	"strings"
	"testing"
)

func TestLoadTemplates_Parse(t *testing.T) {
	t.Parallel()

	if _, err := loadTemplates(); err != nil {
		t.Fatalf("loadTemplates error: %v", err)
	}
}

func TestOverviewLiveStatusScriptHasLocalDifficultyFormatter(t *testing.T) {
	t.Parallel()

	assets, err := newUIAssetLoader()
	if err != nil {
		t.Fatalf("newUIAssetLoader error: %v", err)
	}
	payload, err := assets.readTemplate("overview.tmpl")
	if err != nil {
		t.Fatalf("read overview template: %v", err)
	}
	html := string(payload)
	marker := "const blockDifficultyEl = document.getElementById('status-block-difficulty');"
	idx := strings.Index(html, marker)
	if idx < 0 {
		t.Fatalf("overview live status script marker not found")
	}
	statusScript := html[idx:]
	end := strings.Index(statusScript, "function formatDuration(seconds)")
	if end < 0 {
		t.Fatalf("overview live status difficulty formatter boundary not found")
	}
	formatterBlock := statusScript[:end]
	if strings.Contains(formatterBlock, "formatDiff(") {
		t.Fatalf("live status formatter must not depend on formatDiff from another script closure")
	}
	if !strings.Contains(formatterBlock, "function formatSmallDifficulty(value)") {
		t.Fatalf("live status formatter is missing small difficulty formatting")
	}
	if !strings.Contains(formatterBlock, "1000000000000000") || !strings.Contains(formatterBlock, "1000000000000") {
		t.Fatalf("live status formatter is missing large difficulty units")
	}
}
