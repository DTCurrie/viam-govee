package viamgovee

import (
	"context"
	"fmt"
	"math"

	govee "github.com/DTCurrie/govee-go"
	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

// GoveeLightBrightness is the model identifier for the govee-light-brightness component.
var GoveeLightBrightness = family.WithModel("govee-light-brightness")

func init() {
	resource.RegisterComponent(toggleswitch.API, GoveeLightBrightness,
		resource.Registration[toggleswitch.Switch, *LightBrightnessConfig]{
			Constructor: newGoveeLightBrightness,
		},
	)
}

// LightBrightnessConfig is the configuration for the govee-light-brightness component.
type LightBrightnessConfig struct {
	APIKey   string `json:"api_key"`
	DeviceID string `json:"device"`
	Model    string `json:"model"`
}

// Validate checks that all required fields are set.
func (cfg *LightBrightnessConfig) Validate(_ string) ([]string, []string, error) {
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

type goveeLightBrightness struct {
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	name   resource.Name
	logger logging.Logger
	cfg    *LightBrightnessConfig

	client  *govee.Client
	lastBri int // last brightness set via positions 2-100, used by position 1 to restore
}

func newGoveeLightBrightness(ctx context.Context, _ resource.Dependencies, rawConf resource.Config, logger logging.Logger) (toggleswitch.Switch, error) {
	conf, err := resource.NativeConfig[*LightBrightnessConfig](rawConf)
	if err != nil {
		return nil, err
	}

	client, device, err := connectToDevice(ctx, conf.APIKey, conf.DeviceID, conf.Model, logger)
	if err != nil {
		return nil, err
	}

	s := &goveeLightBrightness{
		name:    rawConf.ResourceName(),
		logger:  logger,
		cfg:     conf,
		client:  client,
		lastBri: 100,
	}

	// Seed lastBri from current device state if available.
	if device.Retrievable {
		if state, err := client.GetDeviceState(ctx, conf.DeviceID, conf.Model); err == nil {
			if state.Brightness > 0 {
				s.lastBri = state.Brightness
			}
		}
	}

	return s, nil
}

func (s *goveeLightBrightness) Name() resource.Name {
	return s.name
}

func (s *goveeLightBrightness) DoCommand(_ context.Context, _ map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

// SetPosition controls on/off and brightness.
// 0 = off. 1 = on at last-set brightness. 2-100 map to brightness levels 1-100%.
func (s *goveeLightBrightness) SetPosition(ctx context.Context, position uint32, _ map[string]interface{}) error {
	if position > 100 {
		return fmt.Errorf("position must be 0-100, got %d", position)
	}

	if position == 0 {
		return s.client.TurnOff(ctx, s.cfg.DeviceID, s.cfg.Model)
	}
	if position == 1 {
		if err := s.client.TurnOn(ctx, s.cfg.DeviceID, s.cfg.Model); err != nil {
			return err
		}
		return s.client.SetBrightness(ctx, s.cfg.DeviceID, s.cfg.Model, s.lastBri)
	}

	if err := s.client.TurnOn(ctx, s.cfg.DeviceID, s.cfg.Model); err != nil {
		return err
	}

	// Map positions 2-100 linearly to brightness 1-100.
	bri := int(math.Round(float64(position-2)/98.0*99.0)) + 1
	s.lastBri = bri
	return s.client.SetBrightness(ctx, s.cfg.DeviceID, s.cfg.Model, bri)
}

// GetPosition returns the current brightness mapped back to a switch position.
func (s *goveeLightBrightness) GetPosition(ctx context.Context, _ map[string]interface{}) (uint32, error) {
	state, err := s.client.GetDeviceState(ctx, s.cfg.DeviceID, s.cfg.Model)
	if err != nil {
		return 0, fmt.Errorf("failed to get device state: %w", err)
	}

	if state.PowerState == "off" || state.Brightness == 0 {
		return 0, nil
	}

	if state.Brightness >= 100 {
		return 100, nil
	}

	// Map brightness 1-100 back to position 2-100.
	pos := uint32(math.Round(float64(state.Brightness-1)/99.0*98.0)) + 2
	if pos > 100 {
		pos = 100
	}
	return pos, nil
}

func (s *goveeLightBrightness) GetNumberOfPositions(_ context.Context, _ map[string]interface{}) (uint32, []string, error) {
	// 0 = off, 1 = on at last brightness, 2-100 = brightness levels
	return 101, nil, nil
}
