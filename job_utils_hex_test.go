package main

import (
	"bytes"
	"testing"
)

func TestHexLUTAcceptsUpperAndLower(t *testing.T) {
	dstLower := make([]byte, 4)
	if err := decodeHexToFixedBytes(dstLower, "deadBEEF"); err != nil {
		t.Fatalf("decode lower/mixed: %v", err)
	}

	dstUpper := make([]byte, 4)
	if err := decodeHexToFixedBytes(dstUpper, "DEADBEEF"); err != nil {
		t.Fatalf("decode upper: %v", err)
	}

	if !bytes.Equal(dstLower, dstUpper) {
		t.Fatalf("mixed-case decode mismatch: lower=%x upper=%x", dstLower, dstUpper)
	}

	gotStrLower, err := parseUint32BEHex("deadbeef")
	if err != nil {
		t.Fatalf("parse lower: %v", err)
	}
	gotStrUpper, err := parseUint32BEHex("DEADBEEF")
	if err != nil {
		t.Fatalf("parse upper: %v", err)
	}
	if gotStrLower != gotStrUpper {
		t.Fatalf("parse mismatch: lower=%08x upper=%08x", gotStrLower, gotStrUpper)
	}

	gotBytesLower, err := parseUint32BEHexBytes([]byte("deadbeef"))
	if err != nil {
		t.Fatalf("parse bytes lower: %v", err)
	}
	gotBytesUpper, err := parseUint32BEHexBytes([]byte("DEADBEEF"))
	if err != nil {
		t.Fatalf("parse bytes upper: %v", err)
	}
	if gotBytesLower != gotBytesUpper {
		t.Fatalf("parse bytes mismatch: lower=%08x upper=%08x", gotBytesLower, gotBytesUpper)
	}
}

func TestParseUint32BEHexPadded(t *testing.T) {
	cases := map[string]uint32{
		"1":        1,
		"00000001": 1,
		"abc":      0xabc,
		"DEADBEEF": 0xdeadbeef,
		"ffffffff": 0xffffffff,
	}

	for input, want := range cases {
		got, err := parseUint32BEHexPadded(input)
		if err != nil {
			t.Fatalf("parseUint32BEHexPadded(%q): %v", input, err)
		}
		if got != want {
			t.Fatalf("parseUint32BEHexPadded(%q)=%08x want=%08x", input, got, want)
		}
	}

	for _, input := range []string{"", "000000001", "zzzzzzzz"} {
		if _, err := parseUint32BEHexPadded(input); err == nil {
			t.Fatalf("parseUint32BEHexPadded(%q) expected error", input)
		}
	}
}
