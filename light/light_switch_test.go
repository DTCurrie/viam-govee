package light

import (
	"context"
	"testing"

	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"

	"github.com/DTCurrie/viam-govee/internal/testutil"
)

func newTestLightSwitch(t *testing.T, stateCapabilities []any) (*goveeLightSwitch, *testutil.TestServer) {
	t.Helper()
	ts := testutil.NewTestServer(t, testutil.TestDeviceList(), stateCapabilities)
	s := &goveeLightSwitch{
		name:   toggleswitch.Named("test-light-switch"),
		logger: logging.NewTestLogger(t),
		cfg: &SwitchConfig{
			APIKey:   "test-key",
			DeviceID: "AA:BB:CC:DD:EE:FF:00:11",
			SKU:      "H6159",
		},
		client: ts.Client,
	}
	return s, ts
}

func TestLightSwitch_SetPosition_Off(t *testing.T) {
	t.Parallel()
	s, ts := newTestLightSwitch(t, nil)

	if err := s.SetPosition(context.Background(), 0, nil); err != nil {
		t.Fatalf("SetPosition(0): %v", err)
	}

	instances := ts.ControlInstances()
	if len(instances) != 1 || instances[0] != "powerSwitch" {
		t.Errorf("expected [powerSwitch], got %v", instances)
	}

	pos, err := s.GetPosition(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPosition: %v", err)
	}
	if pos != 0 {
		t.Errorf("position = %d, want 0", pos)
	}
}

func TestLightSwitch_SetPosition_On(t *testing.T) {
	t.Parallel()
	s, ts := newTestLightSwitch(t, nil)

	if err := s.SetPosition(context.Background(), 1, nil); err != nil {
		t.Fatalf("SetPosition(1): %v", err)
	}

	instances := ts.ControlInstances()
	if len(instances) != 1 || instances[0] != "powerSwitch" {
		t.Errorf("expected [powerSwitch], got %v", instances)
	}

	pos, err := s.GetPosition(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPosition: %v", err)
	}
	if pos != 1 {
		t.Errorf("position = %d, want 1", pos)
	}
}

func TestLightSwitch_SetPosition_Invalid(t *testing.T) {
	t.Parallel()
	s, _ := newTestLightSwitch(t, nil)

	if err := s.SetPosition(context.Background(), 2, nil); err == nil {
		t.Error("expected error for position 2")
	}
}

func TestLightSwitch_GetPosition_LiveState_On(t *testing.T) {
	t.Parallel()
	s, _ := newTestLightSwitch(t, testutil.TestStateOn(80, 0, 0, 0))

	pos, err := s.GetPosition(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPosition: %v", err)
	}
	if pos != 1 {
		t.Errorf("position = %d, want 1", pos)
	}
}

func TestLightSwitch_GetPosition_LiveState_Off(t *testing.T) {
	t.Parallel()
	s, _ := newTestLightSwitch(t, testutil.TestStateOff())

	pos, err := s.GetPosition(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPosition: %v", err)
	}
	if pos != 0 {
		t.Errorf("position = %d, want 0", pos)
	}
}

func TestLightSwitch_GetPosition_FallsBackToLocal(t *testing.T) {
	t.Parallel()
	s, ts := newTestLightSwitch(t, []any{})

	if err := s.SetPosition(context.Background(), 1, nil); err != nil {
		t.Fatalf("SetPosition(1): %v", err)
	}
	ts.ClearControls()

	pos, err := s.GetPosition(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPosition: %v", err)
	}
	if pos != 1 {
		t.Errorf("fallback position = %d, want 1", pos)
	}
}

func TestLightSwitch_GetNumberOfPositions(t *testing.T) {
	t.Parallel()
	s, _ := newTestLightSwitch(t, nil)
	n, labels, err := s.GetNumberOfPositions(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetNumberOfPositions: %v", err)
	}
	if n != 2 {
		t.Errorf("positions = %d, want 2", n)
	}
	if len(labels) != 2 || labels[0] != "off" || labels[1] != "on" {
		t.Errorf("labels = %v, want [off on]", labels)
	}
}

func TestSwitchConfig_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cfg     SwitchConfig
		wantErr bool
	}{
		{SwitchConfig{}, true},
		{SwitchConfig{APIKey: "k"}, true},
		{SwitchConfig{APIKey: "k", DeviceID: "d"}, true},
		{SwitchConfig{APIKey: "k", DeviceID: "d", SKU: "s"}, false},
	}
	for _, tt := range tests {
		_, _, err := tt.cfg.Validate("")
		if (err != nil) != tt.wantErr {
			t.Errorf("Validate(%+v) error = %v, wantErr = %v", tt.cfg, err, tt.wantErr)
		}
	}
}
