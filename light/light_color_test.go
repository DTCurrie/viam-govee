package light

import (
	"context"
	"testing"

	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"

	"github.com/DTCurrie/viam-govee/internal/testutil"
)

func newTestColor(t *testing.T, channel string, stateCapabilities []any) (*goveeLightColor, *testutil.TestServer) {
	t.Helper()
	ts := testutil.NewTestServer(t, testutil.TestDeviceList(), stateCapabilities)
	s := &goveeLightColor{
		name:   toggleswitch.Named("test-color"),
		logger: logging.NewTestLogger(t),
		cfg: &ColorConfig{
			APIKey:   "test-key",
			DeviceID: "AA:BB:CC:DD:EE:FF:00:11",
			SKU:      "H6159",
			Channel:  channel,
		},
		client: ts.Client,
	}
	return s, ts
}

func TestColor_SetPosition_RedChannel(t *testing.T) {
	t.Parallel()
	s, ts := newTestColor(t, "red", testutil.TestStateOn(80, 0, 100, 50))

	if err := s.SetPosition(context.Background(), 200, nil); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	instances := ts.ControlInstances()
	if len(instances) != 2 || instances[0] != "powerSwitch" || instances[1] != "colorRgb" {
		t.Errorf("expected [powerSwitch, colorRgb], got %v", instances)
	}

	calls := ts.Controls()
	colorCall := calls[1]
	var packed int
	if err := testutil.ParseJSON(colorCall.Value, &packed); err != nil {
		t.Fatalf("failed to parse color value: %v", err)
	}
	r := (packed >> 16) & 0xFF
	g := (packed >> 8) & 0xFF
	b := packed & 0xFF
	if r != 200 {
		t.Errorf("red = %d, want 200", r)
	}
	if g != 100 {
		t.Errorf("green = %d, want 100 (preserved)", g)
	}
	if b != 50 {
		t.Errorf("blue = %d, want 50 (preserved)", b)
	}
}

func TestColor_SetPosition_GreenChannel(t *testing.T) {
	t.Parallel()
	s, ts := newTestColor(t, "green", testutil.TestStateOn(80, 100, 0, 50))

	if err := s.SetPosition(context.Background(), 128, nil); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	calls := ts.Controls()
	colorCall := calls[len(calls)-1]
	var packed int
	if err := testutil.ParseJSON(colorCall.Value, &packed); err != nil {
		t.Fatalf("failed to parse color value: %v", err)
	}
	r := (packed >> 16) & 0xFF
	g := (packed >> 8) & 0xFF
	b := packed & 0xFF
	if r != 100 {
		t.Errorf("red = %d, want 100 (preserved)", r)
	}
	if g != 128 {
		t.Errorf("green = %d, want 128", g)
	}
	if b != 50 {
		t.Errorf("blue = %d, want 50 (preserved)", b)
	}
}

func TestColor_SetPosition_AllZeroTurnsOff(t *testing.T) {
	t.Parallel()
	s, ts := newTestColor(t, "red", testutil.TestStateOn(80, 0, 0, 0))

	if err := s.SetPosition(context.Background(), 0, nil); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	instances := ts.ControlInstances()
	if len(instances) != 1 || instances[0] != "powerSwitch" {
		t.Errorf("expected [powerSwitch] (off), got %v", instances)
	}
}

func TestColor_SetPosition_TurnsOnDevice(t *testing.T) {
	t.Parallel()
	s, ts := newTestColor(t, "red", testutil.TestStateOff())

	if err := s.SetPosition(context.Background(), 128, nil); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}

	instances := ts.ControlInstances()
	if len(instances) < 2 || instances[0] != "powerSwitch" {
		t.Errorf("expected powerSwitch as first command, got %v", instances)
	}
}

func TestColor_SetPosition_Invalid(t *testing.T) {
	t.Parallel()
	s, _ := newTestColor(t, "red", testutil.TestStateOn(80, 0, 0, 0))

	if err := s.SetPosition(context.Background(), 256, nil); err == nil {
		t.Error("expected error for position 256")
	}
}

func TestColor_SetPosition_StateAPIError(t *testing.T) {
	t.Parallel()
	s, ts := newTestColor(t, "red", testutil.TestStateOn(80, 0, 0, 0))
	ts.StateError = true

	if err := s.SetPosition(context.Background(), 128, nil); err == nil {
		t.Error("expected error when state API fails")
	}
}

func TestColor_SetPosition_ControlError_OnSetColor(t *testing.T) {
	t.Parallel()
	s, ts := newTestColor(t, "red", testutil.TestStateOn(80, 0, 100, 50))
	ts.ControlFailOn = 2 // TurnOn succeeds (call 1), SetColor fails (call 2)

	if err := s.SetPosition(context.Background(), 128, nil); err == nil {
		t.Error("expected error when SetColor fails")
	}

	if len(ts.Controls()) != 1 {
		t.Errorf("expected 1 successful control call (TurnOn), got %d", len(ts.Controls()))
	}
}

func TestColor_GetPosition(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			s, _ := newTestColor(t, tt.channel, testutil.TestStateOn(80, tt.r, tt.g, tt.b))
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
	t.Parallel()
	s, _ := newTestColor(t, "red", testutil.TestStateOff())
	pos, err := s.GetPosition(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPosition: %v", err)
	}
	if pos != 0 {
		t.Errorf("GetPosition = %d, want 0", pos)
	}
}

func TestColor_GetNumberOfPositions(t *testing.T) {
	t.Parallel()
	s, _ := newTestColor(t, "red", testutil.TestStateOn(80, 0, 0, 0))
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
	t.Parallel()
	tests := []struct {
		cfg     ColorConfig
		wantErr bool
	}{
		{ColorConfig{}, true},
		{ColorConfig{APIKey: "k", DeviceID: "d", SKU: "s"}, true},
		{ColorConfig{APIKey: "k", DeviceID: "d", SKU: "s", Channel: "invalid"}, true},
		{ColorConfig{APIKey: "k", DeviceID: "d", SKU: "s", Channel: "red"}, false},
		{ColorConfig{APIKey: "k", DeviceID: "d", SKU: "s", Channel: "green"}, false},
		{ColorConfig{APIKey: "k", DeviceID: "d", SKU: "s", Channel: "blue"}, false},
	}
	for _, tt := range tests {
		_, _, err := tt.cfg.Validate("")
		if (err != nil) != tt.wantErr {
			t.Errorf("Validate(%+v) error = %v, wantErr = %v", tt.cfg, err, tt.wantErr)
		}
	}
}
