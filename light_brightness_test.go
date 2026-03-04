package viamgovee

import (
	"context"
	"testing"

	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"
)

func newTestBrightness(t *testing.T, stateProps []any) (*goveeLightBrightness, *testServer) {
	t.Helper()
	ts := newTestServer(t, testDeviceList(), stateProps)
	s := &goveeLightBrightness{
		name:   toggleswitch.Named("test-brightness"),
		logger: logging.NewTestLogger(t),
		cfg: &LightBrightnessConfig{
			APIKey:   "test-key",
			DeviceID: "AA:BB:CC:DD:EE:FF:00:11",
			Model:    "H6159",
		},
		client:  ts.Client,
		lastBri: 100,
	}
	return s, ts
}

func TestBrightness_SetPosition_Off(t *testing.T) {
	s, ts := newTestBrightness(t, testStateOn(80, 255, 0, 0))

	if err := s.SetPosition(context.Background(), 0, nil); err != nil {
		t.Fatalf("SetPosition(0): %v", err)
	}

	names := ts.ControlNames()
	if len(names) != 1 || names[0] != "turn" {
		t.Errorf("expected [turn], got %v", names)
	}
}

func TestBrightness_SetPosition_OnAtLastBrightness(t *testing.T) {
	s, ts := newTestBrightness(t, testStateOn(50, 0, 0, 0))
	s.lastBri = 75

	if err := s.SetPosition(context.Background(), 1, nil); err != nil {
		t.Fatalf("SetPosition(1): %v", err)
	}

	names := ts.ControlNames()
	if len(names) != 2 || names[0] != "turn" || names[1] != "brightness" {
		t.Errorf("expected [turn, brightness], got %v", names)
	}
}

func TestBrightness_SetPosition_TurnsOnDevice(t *testing.T) {
	s, ts := newTestBrightness(t, testStateOff())

	if err := s.SetPosition(context.Background(), 50, nil); err != nil {
		t.Fatalf("SetPosition(50): %v", err)
	}

	names := ts.ControlNames()
	if len(names) < 2 || names[0] != "turn" {
		t.Errorf("expected TurnOn as first command, got %v", names)
	}
}

func TestBrightness_SetPosition_InvalidPosition(t *testing.T) {
	s, _ := newTestBrightness(t, testStateOn(80, 0, 0, 0))

	if err := s.SetPosition(context.Background(), 101, nil); err == nil {
		t.Error("expected error for position 101")
	}
}

func TestBrightness_MappingForward(t *testing.T) {
	tests := []struct {
		position uint32
		wantBri  int
	}{
		{2, 1},
		{51, 51},
		{100, 100},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			s, ts := newTestBrightness(t, testStateOn(80, 0, 0, 0))
			if err := s.SetPosition(context.Background(), tt.position, nil); err != nil {
				t.Fatalf("SetPosition(%d): %v", tt.position, err)
			}

			calls := ts.Controls()
			// Last call should be SetBrightness.
			last := calls[len(calls)-1]
			if last.Name != "brightness" {
				t.Fatalf("last command = %q, want brightness", last.Name)
			}
			var bri int
			if err := parseJSON(last.Value, &bri); err != nil {
				t.Fatalf("failed to parse brightness value: %v", err)
			}
			if bri != tt.wantBri {
				t.Errorf("position %d → brightness %d, want %d", tt.position, bri, tt.wantBri)
			}
		})
	}
}

func TestBrightness_MappingReverse(t *testing.T) {
	tests := []struct {
		brightness int
		wantPos    uint32
	}{
		{0, 0},   // off
		{1, 2},   // minimum
		{51, 51}, // midpoint (round-trips with forward mapping)
		{100, 100},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			state := testStateOn(tt.brightness, 0, 0, 0)
			if tt.brightness == 0 {
				state = testStateOff()
			}
			s, _ := newTestBrightness(t, state)

			pos, err := s.GetPosition(context.Background(), nil)
			if err != nil {
				t.Fatalf("GetPosition: %v", err)
			}
			if pos != tt.wantPos {
				t.Errorf("brightness %d → position %d, want %d", tt.brightness, pos, tt.wantPos)
			}
		})
	}
}

func TestBrightness_GetNumberOfPositions(t *testing.T) {
	s, _ := newTestBrightness(t, testStateOn(80, 0, 0, 0))
	n, labels, err := s.GetNumberOfPositions(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetNumberOfPositions: %v", err)
	}
	if n != 101 {
		t.Errorf("positions = %d, want 101", n)
	}
	if labels != nil {
		t.Errorf("labels should be nil, got %v", labels)
	}
}

func TestBrightnessConfig_Validate(t *testing.T) {
	tests := []struct {
		cfg     LightBrightnessConfig
		wantErr bool
	}{
		{LightBrightnessConfig{}, true},
		{LightBrightnessConfig{APIKey: "k"}, true},
		{LightBrightnessConfig{APIKey: "k", DeviceID: "d"}, true},
		{LightBrightnessConfig{APIKey: "k", DeviceID: "d", Model: "m"}, false},
	}
	for _, tt := range tests {
		_, _, err := tt.cfg.Validate("")
		if (err != nil) != tt.wantErr {
			t.Errorf("Validate(%+v) error = %v, wantErr = %v", tt.cfg, err, tt.wantErr)
		}
	}
}
