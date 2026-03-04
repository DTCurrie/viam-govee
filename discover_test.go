package viamgovee

import (
	"context"
	"testing"

	"go.viam.com/rdk/components/sensor"
	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/services/discovery"
)

func TestSanitizeName(t *testing.T) {
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
			got := sanitizeName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDiscoverGovee(t *testing.T) {
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

	// Device 0 (H6159): supports turn, brightness, color, colorTem; retrievable.
	// Expect: 1 brightness switch + 1 sensor + 3 color switches = 5.
	// Device 1 (H5081): supports turn only; not retrievable.
	// Expect: 1 brightness switch = 1.
	// Plus 1 mode switch for color devices.
	// Total: 5 + 1 + 1 = 7.
	if len(configs) != 7 {
		t.Fatalf("expected 7 configs, got %d", len(configs))
		for i, c := range configs {
			t.Logf("  [%d] name=%s api=%s model=%s", i, c.Name, c.API, c.Model)
		}
	}

	// Check that we got the expected types.
	var brightness, sensors, colors, modes int
	for _, c := range configs {
		switch {
		case c.API == toggleswitch.API && c.Model == GoveeLightBrightness:
			brightness++
		case c.API == sensor.API && c.Model == GoveeLightSensor:
			sensors++
		case c.API == toggleswitch.API && c.Model == GoveeLightColor:
			colors++
		case c.API == toggleswitch.API && c.Model == GoveeLightMode:
			modes++
		}
	}

	if brightness != 2 {
		t.Errorf("brightness switches = %d, want 2", brightness)
	}
	if sensors != 1 {
		t.Errorf("sensors = %d, want 1", sensors)
	}
	if colors != 3 {
		t.Errorf("color switches = %d, want 3", colors)
	}
	if modes != 1 {
		t.Errorf("mode switches = %d, want 1", modes)
	}
}

func TestDiscoverGovee_NoDevices(t *testing.T) {
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
	cfg := &DiscoveryConfig{APIKey: ""}
	if _, _, err := cfg.Validate(""); err == nil {
		t.Error("expected error for empty api_key")
	}

	cfg.APIKey = "valid-key"
	if _, _, err := cfg.Validate(""); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
