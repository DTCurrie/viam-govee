package viamgovee

import (
	"context"
	"fmt"
	"strings"

	govee "github.com/DTCurrie/govee-go"
	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

// GoveeLightSensor is the model identifier for the govee-light-sensor component.
var GoveeLightSensor = family.WithModel("govee-light-sensor")

func init() {
	resource.RegisterComponent(sensor.API, GoveeLightSensor,
		resource.Registration[sensor.Sensor, *LightSensorConfig]{
			Constructor: newGoveeLightSensor,
		},
	)
}

// LightSensorConfig is the configuration for the govee-light-sensor component.
type LightSensorConfig struct {
	APIKey   string `json:"api_key"`
	DeviceID string `json:"device"`
	Model    string `json:"model"`
}

// Validate checks that all required fields are set.
func (cfg *LightSensorConfig) Validate(_ string) ([]string, []string, error) {
	if cfg.APIKey == "" {
		return nil, nil, fmt.Errorf("api_key is required")
	}
	if cfg.DeviceID == "" {
		return nil, nil, fmt.Errorf("device (MAC address) is required")
	}
	if cfg.Model == "" {
		return nil, nil, fmt.Errorf("model is required")
	}
	return nil, nil, nil
}

type goveeLightSensor struct {
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	name   resource.Name
	logger logging.Logger
	cfg    *LightSensorConfig

	client *govee.Client
	device *govee.Device
}

func newGoveeLightSensor(ctx context.Context, _ resource.Dependencies, rawConf resource.Config, logger logging.Logger) (sensor.Sensor, error) {
	conf, err := resource.NativeConfig[*LightSensorConfig](rawConf)
	if err != nil {
		return nil, err
	}

	client, device, err := connectToDevice(ctx, conf.APIKey, conf.DeviceID, conf.Model, logger)
	if err != nil {
		return nil, err
	}

	return &goveeLightSensor{
		name:   rawConf.ResourceName(),
		logger: logger,
		cfg:    conf,
		client: client,
		device: device,
	}, nil
}

func (s *goveeLightSensor) Name() resource.Name {
	return s.name
}

func (s *goveeLightSensor) DoCommand(_ context.Context, _ map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

// Readings returns all available information about the Govee device: static
// metadata from the device list and live state from the state query.
func (s *goveeLightSensor) Readings(ctx context.Context, _ map[string]interface{}) (map[string]interface{}, error) {
	readings := map[string]interface{}{
		// Static metadata from device list
		"device_name":  s.device.DeviceName,
		"model":        s.device.Model,
		"device_id":    s.device.DeviceID,
		"controllable": s.device.Controllable,
		"retrievable":  s.device.Retrievable,
		"support_cmds": strings.Join(s.device.SupportCmds, ","),
	}

	if !s.device.Retrievable {
		readings["note"] = "device is not retrievable; live state unavailable"
		return readings, nil
	}

	state, err := s.client.GetDeviceState(ctx, s.cfg.DeviceID, s.cfg.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to get device state: %w", err)
	}

	readings["online"] = state.Online
	readings["power_state"] = state.PowerState
	readings["brightness"] = state.Brightness

	if state.Color != nil {
		readings["red"] = state.Color.R
		readings["green"] = state.Color.G
		readings["blue"] = state.Color.B
	} else {
		readings["red"] = 0
		readings["green"] = 0
		readings["blue"] = 0
	}

	if state.ColorTemp > 0 {
		readings["color_temp"] = state.ColorTemp
	}

	return readings, nil
}
