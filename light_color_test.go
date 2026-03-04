package viamgovee

import (
	"context"
	"testing"

	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"
)

func newTestColor(t *testing.T, channel string, stateProps []any) (*goveeLightColor, *testServer) {
	t.Helper()
	ts := newTestServer(t, testDeviceList(), stateProps)
	s := &goveeLightColor{
		name:   toggleswitch.Named("test-color"),
		logger: logging.NewTestLogger(t),
		cfg: &LightColorConfig{
			APIKey:   "test-key",
			DeviceID: "AA:BB:CC:DD:EE:FF:00:11",
			Model:    "H6159",
			Channel:  channel,
		},
		client: ts.Client,
	}
	return s, ts
}

func TestColor_SetPosition_RedChannel(t *testing.T) {
	s, ts := newTestColor(t, "red", testStateOn(80, 0, 100, 50))

	if err := s.SetPosition(context.Background(), 200, nil); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	names := ts.ControlNames()
	if len(names) != 2 || names[0] != "turn" || names[1] != "color" {
		t.Errorf("expected [turn, color], got %v", names)
	}

	calls := ts.Controls()
	colorCall := calls[1]
	var color map[string]any
	if err := parseJSON(colorCall.Value, &color); err != nil {
		t.Fatalf("failed to parse color: %v", err)
	}
	if color["r"] != float64(200) {
		t.Errorf("red = %v, want 200", color["r"])
	}
	if color["g"] != float64(100) {
		t.Errorf("green = %v, want 100 (preserved)", color["g"])
	}
	if color["b"] != float64(50) {
		t.Errorf("blue = %v, want 50 (preserved)", color["b"])
	}
}

func TestColor_SetPosition_GreenChannel(t *testing.T) {
	s, ts := newTestColor(t, "green", testStateOn(80, 100, 0, 50))

	if err := s.SetPosition(context.Background(), 128, nil); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	calls := ts.Controls()
	colorCall := calls[len(calls)-1]
	var color map[string]any
	if err := parseJSON(colorCall.Value, &color); err != nil {
		t.Fatalf("failed to parse color: %v", err)
	}
	if color["r"] != float64(100) {
		t.Errorf("red = %v, want 100 (preserved)", color["r"])
	}
	if color["g"] != float64(128) {
		t.Errorf("green = %v, want 128", color["g"])
	}
}

func TestColor_SetPosition_AllZeroTurnsOff(t *testing.T) {
	s, ts := newTestColor(t, "red", testStateOn(80, 0, 0, 0))

	if err := s.SetPosition(context.Background(), 0, nil); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	names := ts.ControlNames()
	if len(names) != 1 || names[0] != "turn" {
		t.Errorf("expected [turn] (off), got %v", names)
	}
}

func TestColor_SetPosition_TurnsOnDevice(t *testing.T) {
	s, ts := newTestColor(t, "red", testStateOff())

	if err := s.SetPosition(context.Background(), 128, nil); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	names := ts.ControlNames()
	if len(names) < 2 || names[0] != "turn" {
		t.Errorf("expected TurnOn as first command, got %v", names)
	}
}

func TestColor_SetPosition_Invalid(t *testing.T) {
	s, _ := newTestColor(t, "red", testStateOn(80, 0, 0, 0))

	if err := s.SetPosition(context.Background(), 256, nil); err == nil {
		t.Error("expected error for position 256")
	}
}

func TestColor_GetPosition(t *testing.T) {
	tests := []struct {
		channel string
		r, g, b int
		wantPos uint32
	}{
		{"red", 200, 100, 50, 200},
		{"green", 200, 100, 50, 100},
		{"blue", 200, 100, 50, 50},
	}
	for _, tt := range tests {
		t.Run(tt.channel, func(t *testing.T) {
			s, _ := newTestColor(t, tt.channel, testStateOn(80, tt.r, tt.g, tt.b))
			pos, err := s.GetPosition(context.Background(), nil)
			if err != nil {
				t.Fatalf("GetPosition: %v", err)
			}
			if pos != tt.wantPos {
				t.Errorf("GetPosition = %d, want %d", pos, tt.wantPos)
			}
		})
	}
}

func TestColor_GetPosition_Off(t *testing.T) {
	s, _ := newTestColor(t, "red", testStateOff())
	pos, err := s.GetPosition(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPosition: %v", err)
	}
	if pos != 0 {
		t.Errorf("GetPosition = %d, want 0", pos)
	}
}

func TestColor_GetNumberOfPositions(t *testing.T) {
	s, _ := newTestColor(t, "red", testStateOn(80, 0, 0, 0))
	n, labels, err := s.GetNumberOfPositions(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetNumberOfPositions: %v", err)
	}
	if n != 256 {
		t.Errorf("positions = %d, want 256", n)
	}
	if labels != nil {
		t.Errorf("labels should be nil")
	}
}

func TestColorConfig_Validate(t *testing.T) {
	tests := []struct {
		cfg     LightColorConfig
		wantErr bool
	}{
		{LightColorConfig{}, true},
		{LightColorConfig{APIKey: "k", DeviceID: "d", Model: "m"}, true},
		{LightColorConfig{APIKey: "k", DeviceID: "d", Model: "m", Channel: "invalid"}, true},
		{LightColorConfig{APIKey: "k", DeviceID: "d", Model: "m", Channel: "red"}, false},
		{LightColorConfig{APIKey: "k", DeviceID: "d", Model: "m", Channel: "green"}, false},
		{LightColorConfig{APIKey: "k", DeviceID: "d", Model: "m", Channel: "blue"}, false},
	}
	for _, tt := range tests {
		_, _, err := tt.cfg.Validate("")
		if (err != nil) != tt.wantErr {
			t.Errorf("Validate(%+v) error = %v, wantErr = %v", tt.cfg, err, tt.wantErr)
		}
	}
}
