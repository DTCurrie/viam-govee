package viamgovee

import (
	"context"
	"testing"

	govee "github.com/DTCurrie/govee-go"
	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
)

func TestSensor_Readings_Retrievable(t *testing.T) {
	ts := newTestServer(t, testDeviceList(), testStateOn(80, 255, 128, 0))

	s := &goveeLightSensor{
		name:   sensor.Named("test-sensor"),
		logger: logging.NewTestLogger(t),
		cfg: &LightSensorConfig{
			APIKey:   "test-key",
			DeviceID: "AA:BB:CC:DD:EE:FF:00:11",
			Model:    "H6159",
		},
		client: ts.Client,
		device: &govee.Device{
			DeviceID:     "AA:BB:CC:DD:EE:FF:00:11",
			Model:        "H6159",
			DeviceName:   "Living Room Light",
			Controllable: true,
			Retrievable:  true,
			SupportCmds:  []string{"turn", "brightness", "color", "colorTem"},
		},
	}

	readings, err := s.Readings(context.Background(), nil)
	if err != nil {
		t.Fatalf("Readings: %v", err)
	}

	// Static metadata.
	if readings["device_name"] != "Living Room Light" {
		t.Errorf("device_name = %v", readings["device_name"])
	}
	if readings["model"] != "H6159" {
		t.Errorf("model = %v", readings["model"])
	}
	if readings["controllable"] != true {
		t.Errorf("controllable = %v", readings["controllable"])
	}
	if readings["retrievable"] != true {
		t.Errorf("retrievable = %v", readings["retrievable"])
	}

	// Live state.
	if readings["online"] != true {
		t.Errorf("online = %v", readings["online"])
	}
	if readings["power_state"] != "on" {
		t.Errorf("power_state = %v", readings["power_state"])
	}
	if readings["brightness"] != 80 {
		t.Errorf("brightness = %v", readings["brightness"])
	}
	if readings["red"] != 255 {
		t.Errorf("red = %v", readings["red"])
	}
	if readings["green"] != 128 {
		t.Errorf("green = %v", readings["green"])
	}
	if readings["blue"] != 0 {
		t.Errorf("blue = %v", readings["blue"])
	}
}

func TestSensor_Readings_NonRetrievable(t *testing.T) {
	ts := newTestServer(t, testDeviceList(), nil)

	s := &goveeLightSensor{
		name:   sensor.Named("test-sensor"),
		logger: logging.NewTestLogger(t),
		cfg: &LightSensorConfig{
			APIKey:   "test-key",
			DeviceID: "11:22:33:44:55:66:77:88",
			Model:    "H5081",
		},
		client: ts.Client,
		device: &govee.Device{
			DeviceID:     "11:22:33:44:55:66:77:88",
			Model:        "H5081",
			DeviceName:   "Smart Plug",
			Controllable: true,
			Retrievable:  false,
			SupportCmds:  []string{"turn"},
		},
	}

	readings, err := s.Readings(context.Background(), nil)
	if err != nil {
		t.Fatalf("Readings: %v", err)
	}

	if readings["device_name"] != "Smart Plug" {
		t.Errorf("device_name = %v", readings["device_name"])
	}
	if readings["note"] == nil {
		t.Error("expected note for non-retrievable device")
	}
	if readings["online"] != nil {
		t.Error("non-retrievable device should not have online field")
	}
}

func TestSensorConfig_Validate(t *testing.T) {
	tests := []struct {
		cfg     LightSensorConfig
		wantErr bool
	}{
		{LightSensorConfig{}, true},
		{LightSensorConfig{APIKey: "k"}, true},
		{LightSensorConfig{APIKey: "k", DeviceID: "d"}, true},
		{LightSensorConfig{APIKey: "k", DeviceID: "d", Model: "m"}, false},
	}
	for _, tt := range tests {
		_, _, err := tt.cfg.Validate("")
		if (err != nil) != tt.wantErr {
			t.Errorf("Validate(%+v) error = %v, wantErr = %v", tt.cfg, err, tt.wantErr)
		}
	}
}
