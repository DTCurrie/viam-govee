package light

import (
	"context"
	"fmt"
	"testing"

	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"

	"github.com/DTCurrie/viam-govee/internal/testutil"
)

func newTestBrightness(t *testing.T, stateCapabilities []any) (*goveeLightBrightness, *testutil.TestServer) {
	t.Helper()
	ts := testutil.NewTestServer(t, testutil.TestDeviceList(), stateCapabilities)
	s := &goveeLightBrightness{
		name:   toggleswitch.Named("test-brightness"),
		logger: logging.NewTestLogger(t),
		cfg: &BrightnessConfig{
			APIKey:   "test-key",
			DeviceID: "AA:BB:CC:DD:EE:FF:00:11",
			SKU:      "H6159",
		},
		client: ts.Client,
	}
	return s, ts
}

func TestBrightness_SetPosition_InvalidPosition(t *testing.T) {
	t.Parallel()
	s, _ := newTestBrightness(t, testutil.TestStateOn(80, 0, 0, 0))

	if err := s.SetPosition(context.Background(), 0, nil); err == nil {
		t.Error("expected error for position 0")
	}
	if err := s.SetPosition(context.Background(), 101, nil); err == nil {
		t.Error("expected error for position 101")
	}
}

func TestBrightness_MappingForward(t *testing.T) {
	t.Parallel()
	tests := []struct {
		position uint32
		wantBri  int
	}{
		{1, 1},
		{50, 50},
		{100, 100},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("position_%d", tt.position), func(t *testing.T) {
			t.Parallel()
			s, ts := newTestBrightness(t, testutil.TestStateOn(80, 0, 0, 0))
			if err := s.SetPosition(context.Background(), tt.position, nil); err != nil {
				t.Fatalf("SetPosition(%d): %v", tt.position, err)
			}

			calls := ts.Controls()
			if len(calls) != 1 || calls[0].Instance != "brightness" {
				t.Fatalf("expected single brightness command, got %v", ts.ControlInstances())
			}
			var bri int
			if err := testutil.ParseJSON(calls[0].Value, &bri); err != nil {
				t.Fatalf("failed to parse brightness value: %v", err)
			}
			if bri != tt.wantBri {
				t.Errorf("position %d → brightness %d, want %d", tt.position, bri, tt.wantBri)
			}
		})
	}
}

func TestBrightness_MappingReverse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		brightness int
		wantPos    uint32
	}{
		{1, 1},
		{50, 50},
		{100, 100},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("brightness_%d", tt.brightness), func(t *testing.T) {
			t.Parallel()
			s, _ := newTestBrightness(t, testutil.TestStateOn(tt.brightness, 0, 0, 0))

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

func TestBrightness_GetPosition_Unavailable(t *testing.T) {
	t.Parallel()
	s, _ := newTestBrightness(t, testutil.TestStateOff())

	pos, err := s.GetPosition(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPosition: %v", err)
	}
	if pos != 0 {
		t.Errorf("expected position 0 when brightness unavailable, got %d", pos)
	}
}

func TestBrightness_GetNumberOfPositions(t *testing.T) {
	t.Parallel()
	s, _ := newTestBrightness(t, testutil.TestStateOn(80, 0, 0, 0))
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

func TestBrightnessConfig_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cfg     BrightnessConfig
		wantErr bool
	}{
		{BrightnessConfig{}, true},
		{BrightnessConfig{APIKey: "k"}, true},
		{BrightnessConfig{APIKey: "k", DeviceID: "d"}, true},
		{BrightnessConfig{APIKey: "k", DeviceID: "d", SKU: "s"}, false},
	}
	for _, tt := range tests {
		_, _, err := tt.cfg.Validate("")
		if (err != nil) != tt.wantErr {
			t.Errorf("Validate(%+v) error = %v, wantErr = %v", tt.cfg, err, tt.wantErr)
		}
	}
}
