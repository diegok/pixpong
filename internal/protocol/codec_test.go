package protocol

import (
	"bytes"
	"testing"
)

func TestCodec_EncodeDecodeRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	codec := NewCodec(&buf)

	original := &Message{
		Type: MsgGameState,
		Payload: GameState{
			Tick: 42,
			Ball: BallState{X: 10.5, Y: 20.3, VX: 1.0, VY: -0.5},
		},
	}

	if err := codec.Encode(original); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := codec.Decode()
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("type mismatch: got %v, want %v", decoded.Type, original.Type)
	}

	state, ok := decoded.Payload.(GameState)
	if !ok {
		t.Fatalf("payload type mismatch")
	}

	if state.Tick != 42 {
		t.Errorf("tick mismatch: got %d, want 42", state.Tick)
	}
}
