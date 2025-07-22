package tftp

import (
	"testing"
)

func TestRRQ_PackUnpack(t *testing.T) {
	filename := "test.txt"
	mode := "octet"

	rrq := &ReadRequest{
		Filename: filename,
		Mode:     mode,
	}
	packet := rrq.Serialize()

	if len(packet) != 2+len(filename)+1+len(mode)+1 {
		t.Fatalf("unexpected packet length: got %d", len(packet))
	}

	parsed, err := ParseRRQ(packet)
	if err != nil {
		t.Fatalf("ParseRRQ failed: %v", err)
	}

	if parsed.Filename != filename {
		t.Errorf("expected filename %q, got %q", filename, parsed.Filename)
	}
	if parsed.Mode != mode {
		t.Errorf("expected mode %q, got %q", mode, parsed.Mode)
	}
}

func TestRRQ_ExactBytes(t *testing.T) {
	// RRQ: filename = "file", mode = "netascii"
	expected := []byte{
		0x00, 0x01, // Opcode: RRQ
		'f', 'i', 'l', 'e',
		0x00, // null
		'n', 'e', 't', 'a', 's', 'c', 'i', 'i',
		0x00, // null
	}

	rrq := &ReadRequest{Filename: "file", Mode: "netascii"}
	got := rrq.Serialize()

	if len(got) != len(expected) {
		t.Fatalf("length mismatch: got %d, expected %d", len(got), len(expected))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("byte %d mismatch: got 0x%02x, expected 0x%02x", i, got[i], expected[i])
		}
	}
}

func TestRRQ_WrongOpcode(t *testing.T) {
	packet := []byte{0x00, 0x02, 'f', 'i', 'l', 'e', 0, 'o', 'c', 't', 'e', 't', 0}

	_, err := ParseRRQ(packet)
	if err == nil {
		t.Fatal("expected error for wrong opcode, got nil")
	}
}

func TestRRQ_TooShort(t *testing.T) {
	packet := []byte{0x00, 0x01}

	_, err := ParseRRQ(packet)
	if err == nil {
		t.Fatal("expected error for short packet")
	}
	if err != ErrPacketTooShort {
		t.Errorf("expected ErrPacketTooShort, got %v", err)
	}
}

func TestDATA_PackUnpack(t *testing.T) {
	block := uint16(42)
	data := []byte("Hello, TFTP!")

	packet := PackDATA(block, data)
	if len(packet) != 4+len(data) {
		t.Fatalf("wrong packet length: %d", len(packet))
	}

	parsed, err := ParseDATA(packet)
	if err != nil {
		t.Fatalf("ParseDATA failed: %v", err)
	}

	if parsed.Block != block {
		t.Errorf("expected block %d, got %d", block, parsed.Block)
	}
	if string(parsed.Data) != string(data) {
		t.Errorf("data mismatch")
	}
}

func TestACK_PackUnpack(t *testing.T) {
	block := uint16(13)

	packet := PackACK(block)
	if len(packet) != 4 {
		t.Fatalf("ACK packet should be 4 bytes, got %d", len(packet))
	}

	parsed, err := ParseACK(packet)
	if err != nil {
		t.Fatalf("ParseACK failed: %v", err)
	}

	if parsed.Block != block {
		t.Errorf("expected block %d, got %d", block, parsed.Block)
	}
}

func TestERROR_PackUnpack(t *testing.T) {
	code := uint16(2)
	msg := "Access violation"

	packet := PackERROR(code, msg)
	parsed, err := ParseERROR(packet)

	if err != nil {
		t.Fatalf("ParseERROR failed: %v", err)
	}

	if parsed.Code != code {
		t.Errorf("expected code %d, got %d", code, parsed.Code)
	}
	if parsed.Message != msg {
		t.Errorf("expected message %q, got %q", msg, parsed.Message)
	}
	if packet[len(packet)-1] != 0 {
		t.Error("ERROR packet must end with null byte")
	}
}
