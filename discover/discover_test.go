package discover

import (
	"context"
	"testing"

	"go.viam.com/rdk/components/sensor"
	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/services/discovery"

	"github.com/DTCurrie/viam-govee/internal/testutil"
	"github.com/DTCurrie/viam-govee/light"
	"github.com/DTCurrie/viam-govee/plug"
)

func TestSanitizeName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"Living Room Light", "Living-Room-Light"},
		{"simple", "simple"},
		{"has  spaces", "has-spaces"},
		{"special!@#chars", "special-chars"},
		{"---leading-trailing---", "leading-trailing"},
		{"MiXeD_CaSe-123", "MiXeD_CaSe-123"},
		{"a/b\\c:d", "a-b-c-d"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := sanitizeName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func newTestServer(t *testing.T, devices []map[string]any, stateCapabilities []any) *testutil.TestServer {
	t.Helper()
	return testutil.NewTestServer(t, devices, stateCapabilities)
}

func testDeviceList() []map[string]any {
	return testutil.TestDeviceList()
}

func TestDiscoverGovee(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t, testDeviceList(), nil)

	svc := &GoveeDiscover{
		name:   discovery.Named("test-discovery"),
		logger: logging.NewTestLogger(t),
		cfg:    &DiscoveryConfig{APIKey: "test-key"},
		client: ts.Client,
	}

	configs, err := svc.DiscoverResources(context.Background(), nil)
	if err != nil {
		t.Fatalf("DiscoverResources: %v", err)
	}

	// Device 0 (H6159, light): supports on_off, brightness, colorRgb, colorTemperatureK, lightScene, diyScene, snapshot, gradientToggle.
	// Expect: 1 light switch + 1 brightness + 1 color-temp + 1 sensor + 3 color + 1 scene + 1 DIY + 1 snapshot = 10.
	// Device 1 (H5081, socket): supports on_off only.
	// Expect: 1 plug switch + 1 plug sensor = 2.
	// Total: 10 + 2 = 12.
	if len(configs) != 12 {
		for i, c := range configs {
			t.Logf("  [%d] name=%s api=%s model=%s", i, c.Name, c.API, c.Model)
		}
		t.Fatalf("expected 12 configs, got %d", len(configs))
	}

	var lightSwitches, brightness, colorTemps, lightSensors, colors, scenes, diyScenes, snapshots, plugSwitches, plugSensors int
	for _, c := range configs {
		switch {
		case c.API == toggleswitch.API && c.Model == light.GoveeLightSwitch:
			lightSwitches++
		case c.API == toggleswitch.API && c.Model == light.GoveeLightBrightness:
			brightness++
		case c.API == toggleswitch.API && c.Model == light.GoveeLightColorTemp:
			colorTemps++
		case c.API == sensor.API && c.Model == light.GoveeLightSensor:
			lightSensors++
		case c.API == toggleswitch.API && c.Model == light.GoveeLightColor:
			colors++
		case c.API == toggleswitch.API && c.Model == light.GoveeLightScene:
			scenes++
		case c.API == toggleswitch.API && c.Model == light.GoveeLightDIYScene:
			diyScenes++
		case c.API == toggleswitch.API && c.Model == light.GoveeLightSnapshot:
			snapshots++
		case c.API == toggleswitch.API && c.Model == plug.GoveePlugSwitch:
			plugSwitches++
		case c.API == sensor.API && c.Model == plug.GoveePlugSensor:
			plugSensors++
		}
	}

	if lightSwitches != 1 {
		t.Errorf("light switches = %d, want 1", lightSwitches)
	}
	if brightness != 1 {
		t.Errorf("brightness switches = %d, want 1", brightness)
	}
	if colorTemps != 1 {
		t.Errorf("color temp switches = %d, want 1", colorTemps)
	}
	if lightSensors != 1 {
		t.Errorf("light sensors = %d, want 1", lightSensors)
	}
	if colors != 3 {
		t.Errorf("color switches = %d, want 3", colors)
	}
	if scenes != 1 {
		t.Errorf("scene switches = %d, want 1", scenes)
	}
	if diyScenes != 1 {
		t.Errorf("DIY scene switches = %d, want 1", diyScenes)
	}
	if snapshots != 1 {
		t.Errorf("snapshot switches = %d, want 1", snapshots)
	}
	if plugSwitches != 1 {
		t.Errorf("plug switches = %d, want 1", plugSwitches)
	}
	if plugSensors != 1 {
		t.Errorf("plug sensors = %d, want 1", plugSensors)
	}
}

func TestDiscoverGovee_NoDevices(t *testing.T) {
	t.Parallel()
	ts := newTestServer(t, []map[string]any{}, nil)

	svc := &GoveeDiscover{
		name:   discovery.Named("test-discovery"),
		logger: logging.NewTestLogger(t),
		cfg:    &DiscoveryConfig{APIKey: "test-key"},
		client: ts.Client,
	}

	configs, err := svc.DiscoverResources(context.Background(), nil)
	if err != nil {
		t.Fatalf("DiscoverResources: %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("expected 0 configs for empty device list, got %d", len(configs))
	}
}

func TestDiscoveryConfig_Validate(t *testing.T) {
	t.Parallel()
	cfg := &DiscoveryConfig{APIKey: ""}
	if _, _, err := cfg.Validate(""); err == nil {
		t.Error("expected error for empty api_key")
	}

	cfg.APIKey = "valid-key"
	if _, _, err := cfg.Validate(""); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
