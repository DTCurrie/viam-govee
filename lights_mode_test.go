package viamgovee

import (
	"context"
	"testing"

	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"
)

func newTestMode(t *testing.T, stateProps []any) (*goveeLightMode, *testServer) {
	t.Helper()
	ts := newTestServer(t, testDeviceList(), stateProps)
	s := &goveeLightMode{
		name:   toggleswitch.Named("test-mode"),
		logger: logging.NewTestLogger(t),
		cfg: &LightModeConfig{
			APIKey: "test-key",
			Daylight: []DeviceRef{
				{DeviceID: "AA:BB:CC:DD:EE:FF:00:11", Model: "H6159"},
			},
			Warm: []DeviceRef{
				{DeviceID: "AA:BB:CC:DD:EE:FF:00:11", Model: "H6159"},
			},
		},
		client:      ts.Client,
		savedStates: make(map[string]*savedState),
	}
	return s, ts
}

func TestMode_SetPosition_Daylight(t *testing.T) {
	s, ts := newTestMode(t, testStateOn(80, 255, 128, 0))

	if err := s.SetPosition(context.Background(), 1, nil); err != nil {
		t.Fatalf("SetPosition(1): %v", err)
	}

	names := ts.ControlNames()
	// Expect: TurnOn, SetColorTemp(6500), SetBrightness(100).
	want := []string{"turn", "colorTem", "brightness"}
	if len(names) != len(want) {
		t.Fatalf("control calls = %v, want %v", names, want)
	}
	for i, n := range want {
		if names[i] != n {
			t.Errorf("call[%d] = %q, want %q", i, names[i], n)
		}
	}

	pos, _ := s.GetPosition(context.Background(), nil)
	if pos != 1 {
		t.Errorf("position = %d, want 1", pos)
	}
}

func TestMode_SetPosition_Warm(t *testing.T) {
	s, ts := newTestMode(t, testStateOn(80, 255, 128, 0))

	if err := s.SetPosition(context.Background(), 2, nil); err != nil {
		t.Fatalf("SetPosition(2): %v", err)
	}

	names := ts.ControlNames()
	want := []string{"turn", "colorTem", "brightness"}
	if len(names) != len(want) {
		t.Fatalf("control calls = %v, want %v", names, want)
	}
	for i, n := range want {
		if names[i] != n {
			t.Errorf("call[%d] = %q, want %q", i, names[i], n)
		}
	}
}

func TestMode_SetPosition_Restore(t *testing.T) {
	s, ts := newTestMode(t, testStateOn(80, 255, 128, 0))

	// Activate a mode first.
	if err := s.SetPosition(context.Background(), 1, nil); err != nil {
		t.Fatalf("SetPosition(1): %v", err)
	}
	ts.ClearControls()

	// Restore.
	if err := s.SetPosition(context.Background(), 0, nil); err != nil {
		t.Fatalf("SetPosition(0): %v", err)
	}

	names := ts.ControlNames()
	// Expect: TurnOn, SetColor (saved color), SetBrightness (saved brightness).
	if len(names) < 2 {
		t.Fatalf("expected at least 2 restore calls, got %v", names)
	}
	if names[0] != "turn" {
		t.Errorf("first restore call = %q, want turn", names[0])
	}

	pos, _ := s.GetPosition(context.Background(), nil)
	if pos != 0 {
		t.Errorf("position after restore = %d, want 0", pos)
	}
}

func TestMode_SetPosition_RestoreOff(t *testing.T) {
	s, ts := newTestMode(t, testStateOff())

	// Activate a mode.
	if err := s.SetPosition(context.Background(), 1, nil); err != nil {
		t.Fatalf("SetPosition(1): %v", err)
	}
	ts.ClearControls()

	// Restore should turn off the device.
	if err := s.SetPosition(context.Background(), 0, nil); err != nil {
		t.Fatalf("SetPosition(0): %v", err)
	}

	names := ts.ControlNames()
	if len(names) != 1 || names[0] != "turn" {
		t.Errorf("expected [turn] (off), got %v", names)
	}
}

func TestMode_SetPosition_Invalid(t *testing.T) {
	s, _ := newTestMode(t, testStateOn(80, 0, 0, 0))

	if err := s.SetPosition(context.Background(), 99, nil); err == nil {
		t.Error("expected error for invalid position")
	}
}

func TestMode_GetNumberOfPositions(t *testing.T) {
	s, _ := newTestMode(t, testStateOn(80, 0, 0, 0))
	n, labels, err := s.GetNumberOfPositions(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetNumberOfPositions: %v", err)
	}
	if n != 3 {
		t.Errorf("positions = %d, want 3", n)
	}
	if len(labels) != 3 {
		t.Fatalf("labels length = %d, want 3", len(labels))
	}
	wantLabels := []string{"none", "daylight", "warm"}
	for i, l := range wantLabels {
		if labels[i] != l {
			t.Errorf("label[%d] = %q, want %q", i, labels[i], l)
		}
	}
}

func TestModeConfig_Validate(t *testing.T) {
	cfg := &LightModeConfig{APIKey: ""}
	if _, _, err := cfg.Validate(""); err == nil {
		t.Error("expected error for empty api_key")
	}

	cfg.APIKey = "valid-key"
	if _, _, err := cfg.Validate(""); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
