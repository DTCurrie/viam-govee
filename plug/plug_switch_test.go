package plug

import (
	"context"
	"testing"

	govee "github.com/DTCurrie/govee-go"
	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"

	"github.com/DTCurrie/viam-govee/internal/testutil"
)

func newTestPlugSwitch(t *testing.T, stateCapabilities []any) (*goveePlugSwitch, *testutil.TestServer) {
	t.Helper()
	ts := testutil.NewTestServer(t, testutil.TestDeviceList(), stateCapabilities)
	s := &goveePlugSwitch{
		name:   toggleswitch.Named("test-plug-switch"),
		logger: logging.NewTestLogger(t),
		cfg: &SwitchConfig{
			APIKey:   "test-key",
			DeviceID: "11:22:33:44:55:66:77:88",
			SKU:      "H5081",
		},
		client: ts.Client,
	}
	return s, ts
}

func testPlugDevice() *govee.Device {
	return &govee.Device{
		DeviceID:   "11:22:33:44:55:66:77:88",
		SKU:        "H5081",
		DeviceName: "Smart Plug",
		Type:       govee.DeviceSocket,
		Capabilities: []govee.Capability{
			{Type: govee.CapabilityOnOff, Instance: "powerSwitch"},
		},
	}
}

func TestPlugSwitch_SetPosition_Off(t *testing.T) {
	t.Parallel()
	s, ts := newTestPlugSwitch(t, nil)

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

func TestPlugSwitch_SetPosition_On(t *testing.T) {
	t.Parallel()
	s, ts := newTestPlugSwitch(t, nil)

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

func TestPlugSwitch_SetPosition_Invalid(t *testing.T) {
	t.Parallel()
	s, _ := newTestPlugSwitch(t, nil)

	if err := s.SetPosition(context.Background(), 2, nil); err == nil {
		t.Error("expected error for position 2")
	}
}

func TestPlugSwitch_GetPosition_LiveState_On(t *testing.T) {
	t.Parallel()
	s, _ := newTestPlugSwitch(t, testutil.TestStateOn(0, 0, 0, 0))

	pos, err := s.GetPosition(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPosition: %v", err)
	}
	if pos != 1 {
		t.Errorf("position = %d, want 1", pos)
	}
}

func TestPlugSwitch_GetPosition_LiveState_Off(t *testing.T) {
	t.Parallel()
	s, _ := newTestPlugSwitch(t, testutil.TestStateOff())

	pos, err := s.GetPosition(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPosition: %v", err)
	}
	if pos != 0 {
		t.Errorf("position = %d, want 0", pos)
	}
}

func TestPlugSwitch_GetPosition_FallsBackToLocal(t *testing.T) {
	t.Parallel()
	s, ts := newTestPlugSwitch(t, []any{})

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

func TestPlugSwitch_GetNumberOfPositions(t *testing.T) {
	t.Parallel()
	s, _ := newTestPlugSwitch(t, nil)
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
