package light

import (
	"context"
	"fmt"
	"testing"

	govee "github.com/DTCurrie/govee-go"
	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"

	"github.com/DTCurrie/viam-govee/internal/testutil"
)

func newTestColorTemp(t *testing.T, stateCapabilities []any) (*goveeLightColorTemp, *testutil.TestServer) {
	t.Helper()
	ts := testutil.NewTestServer(t, testutil.TestDeviceList(), stateCapabilities)
	s := &goveeLightColorTemp{
		name:   toggleswitch.Named("test-color-temp"),
		logger: logging.NewTestLogger(t),
		cfg: &ColorTempConfig{
			APIKey:   "test-key",
			DeviceID: "AA:BB:CC:DD:EE:FF:00:11",
			SKU:      "H6159",
		},
		client: ts.Client,
		minK:   2000,
		maxK:   9000,
	}
	return s, ts
}

func TestColorTemp_SetPosition_InvalidPosition(t *testing.T) {
	t.Parallel()
	s, _ := newTestColorTemp(t, testutil.TestStateOn(80, 0, 0, 0))

	if err := s.SetPosition(context.Background(), 0, nil); err == nil {
		t.Error("expected error for position 0")
	}
	if err := s.SetPosition(context.Background(), 101, nil); err == nil {
		t.Error("expected error for position 101")
	}
}

func TestColorTemp_MappingForward(t *testing.T) {
	t.Parallel()
	tests := []struct {
		position   uint32
		wantKelvin int
	}{
		{1, 2000},
		{50, 5465},
		{100, 9000},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("position_%d_to_%dK", tt.position, tt.wantKelvin), func(t *testing.T) {
			t.Parallel()
			s, ts := newTestColorTemp(t, testutil.TestStateOn(80, 0, 0, 0))
			if err := s.SetPosition(context.Background(), tt.position, nil); err != nil {
				t.Fatalf("SetPosition(%d): %v", tt.position, err)
			}

			calls := ts.Controls()
			if len(calls) != 1 || calls[0].Instance != "colorTemperatureK" {
				t.Fatalf("expected single colorTemperatureK command, got %v", ts.ControlInstances())
			}
			var kelvin int
			if err := testutil.ParseJSON(calls[0].Value, &kelvin); err != nil {
				t.Fatalf("failed to parse kelvin value: %v", err)
			}
			if kelvin != tt.wantKelvin {
				t.Errorf("position %d → kelvin %d, want %d", tt.position, kelvin, tt.wantKelvin)
			}
		})
	}
}

func TestColorTemp_MappingReverse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		kelvin  int
		wantPos uint32
	}{
		{2000, 1},
		{5465, 50},
		{9000, 100},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%dK_to_position_%d", tt.kelvin, tt.wantPos), func(t *testing.T) {
			t.Parallel()
			state := testStateOnWithColorTemp(tt.kelvin, true)
			s, _ := newTestColorTemp(t, state)

			pos, err := s.GetPosition(context.Background(), nil)
			if err != nil {
				t.Fatalf("GetPosition: %v", err)
			}
			if pos != tt.wantPos {
				t.Errorf("kelvin %d → position %d, want %d", tt.kelvin, pos, tt.wantPos)
			}
		})
	}
}

func TestColorTemp_GetPosition_Unavailable(t *testing.T) {
	t.Parallel()
	s, _ := newTestColorTemp(t, testutil.TestStateOff())

	pos, err := s.GetPosition(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPosition: %v", err)
	}
	if pos != 0 {
		t.Errorf("expected position 0 (sentinel) when color temp unavailable, got %d", pos)
	}
}

func TestColorTemp_GetNumberOfPositions(t *testing.T) {
	t.Parallel()
	s, _ := newTestColorTemp(t, testutil.TestStateOn(80, 0, 0, 0))
	n, labels, err := s.GetNumberOfPositions(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetNumberOfPositions: %v", err)
	}
	if n != 100 {
		t.Errorf("positions = %d, want 100", n)
	}
	if labels != nil {
		t.Errorf("labels should be nil, got %v", labels)
	}
}

func TestColorTempConfig_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cfg     ColorTempConfig
		wantErr bool
	}{
		{ColorTempConfig{}, true},
		{ColorTempConfig{APIKey: "k"}, true},
		{ColorTempConfig{APIKey: "k", DeviceID: "d"}, true},
		{ColorTempConfig{APIKey: "k", DeviceID: "d", SKU: "s"}, false},
	}
	for _, tt := range tests {
		_, _, err := tt.cfg.Validate("")
		if (err != nil) != tt.wantErr {
			t.Errorf("Validate(%+v) error = %v, wantErr = %v", tt.cfg, err, tt.wantErr)
		}
	}
}

func TestParseColorTempRange(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		device  *govee.Device
		wantMin int
		wantMax int
		wantErr bool
	}{
		{
			name: "valid range",
			device: &govee.Device{Capabilities: []govee.Capability{
				{
					Type:       govee.CapabilityColorSetting,
					Instance:   "colorTemperatureK",
					Parameters: []byte(`{"range":{"min":2000,"max":9000}}`),
				},
			}},
			wantMin: 2000,
			wantMax: 9000,
		},
		{
			name:    "no capability",
			device:  &govee.Device{},
			wantErr: true,
		},
		{
			name: "invalid range (min=0)",
			device: &govee.Device{Capabilities: []govee.Capability{
				{
					Type:       govee.CapabilityColorSetting,
					Instance:   "colorTemperatureK",
					Parameters: []byte(`{"range":{"min":0,"max":9000}}`),
				},
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			min, max, err := parseColorTempRange(tt.device)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseColorTempRange() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if err == nil {
				if min != tt.wantMin {
					t.Errorf("minK = %d, want %d", min, tt.wantMin)
				}
				if max != tt.wantMax {
					t.Errorf("maxK = %d, want %d", max, tt.wantMax)
				}
			}
		})
	}
}

// testStateOnWithColorTemp returns capability states for a device with a specific color temperature.
func testStateOnWithColorTemp(kelvin int, on bool) []any {
	powerVal := 0
	if on {
		powerVal = 1
	}
	return []any{
		map[string]any{
			"type":     "devices.capabilities.online",
			"instance": "online",
			"state":    map[string]any{"value": true},
		},
		map[string]any{
			"type":     "devices.capabilities.on_off",
			"instance": "powerSwitch",
			"state":    map[string]any{"value": powerVal},
		},
		map[string]any{
			"type":     "devices.capabilities.color_setting",
			"instance": "colorTemperatureK",
			"state":    map[string]any{"value": kelvin},
		},
	}
}
