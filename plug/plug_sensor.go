package plug

import (
	"context"
	"encoding/json"
	"fmt"

	govee "github.com/DTCurrie/govee-go"
	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"

	common "github.com/DTCurrie/viam-govee/internal/common"
)

// GoveePlugSensor is the model identifier for the govee-plug-sensor component.
var GoveePlugSensor = common.Family.WithModel("govee-plug-sensor")

func init() {
	resource.RegisterComponent(sensor.API, GoveePlugSensor,
		resource.Registration[sensor.Sensor, *SensorConfig]{
			Constructor: newGoveePlugSensor,
		},
	)
}

// SensorConfig is the configuration for the govee-plug-sensor component.
type SensorConfig struct {
	APIKey   string `json:"api_key"`
	DeviceID string `json:"device"`
	SKU      string `json:"sku"`
}

// Validate checks that all required fields are set.
func (cfg *SensorConfig) Validate(_ string) ([]string, []string, error) {
	if cfg.APIKey == "" {
		return nil, nil, fmt.Errorf("api_key is required")
	}
	if cfg.DeviceID == "" {
		return nil, nil, fmt.Errorf("device (MAC address) is required")
	}
	if cfg.SKU == "" {
		return nil, nil, fmt.Errorf("sku is required")
	}
	return nil, nil, nil
}

type goveePlugSensor struct {
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	name   resource.Name
	logger logging.Logger
	cfg    *SensorConfig

	client *govee.Client
	device *govee.Device
}

func newGoveePlugSensor(ctx context.Context, _ resource.Dependencies, rawConf resource.Config, logger logging.Logger) (sensor.Sensor, error) {
	conf, err := resource.NativeConfig[*SensorConfig](rawConf)
	if err != nil {
		return nil, err
	}

	client, device, err := common.ConnectToDevice(ctx, conf.APIKey, conf.DeviceID, conf.SKU, logger)
	if err != nil {
		return nil, err
	}

	return &goveePlugSensor{
		name:   rawConf.ResourceName(),
		logger: logger,
		cfg:    conf,
		client: client,
		device: device,
	}, nil
}

func (s *goveePlugSensor) Name() resource.Name {
	return s.name
}

func (s *goveePlugSensor) DoCommand(_ context.Context, _ map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

// Readings returns device metadata and live power state when available.
func (s *goveePlugSensor) Readings(ctx context.Context, _ map[string]interface{}) (map[string]interface{}, error) {
	caps := make([]string, 0, len(s.device.Capabilities))
	for _, c := range s.device.Capabilities {
		caps = append(caps, c.Type+"/"+c.Instance)
	}

	readings := map[string]interface{}{
		"device_name":  s.device.DeviceName,
		"sku":          s.device.SKU,
		"device_id":    s.device.DeviceID,
		"device_type":  s.device.Type,
		"capabilities": caps,
	}

	state, err := s.client.GetDeviceState(ctx, s.cfg.SKU, s.cfg.DeviceID)
	if err != nil {
		readings["note"] = "live state unavailable: " + err.Error()
		return readings, nil
	}

	readings["online"] = state.IsOnline()

	if on, ok := state.PowerState(); ok {
		readings["power_state"] = on
	}

	// Add toggle readings for any toggle capabilities reported by the device.
	for _, cap := range s.device.Capabilities {
		if cap.Type != govee.CapabilityToggle {
			continue
		}
		toggleState := state.FindState(govee.CapabilityToggle, cap.Instance)
		if toggleState == nil || toggleState.State == nil {
			continue
		}
		var v int
		if err := json.Unmarshal(toggleState.State.Value, &v); err != nil {
			continue
		}
		readings["toggle_"+cap.Instance] = v == 1
	}

	return readings, nil
}
