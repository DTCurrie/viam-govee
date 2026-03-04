package viamgovee

import (
	"context"
	"fmt"

	govee "github.com/DTCurrie/govee-go"
	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

// GoveeLightColor is the model identifier for the govee-light-color component.
var GoveeLightColor = family.WithModel("govee-light-color")

func init() {
	resource.RegisterComponent(toggleswitch.API, GoveeLightColor,
		resource.Registration[toggleswitch.Switch, *LightColorConfig]{
			Constructor: newGoveeLightColor,
		},
	)
}

// LightColorConfig is the configuration for the govee-light-color component.
type LightColorConfig struct {
	APIKey   string `json:"api_key"`
	DeviceID string `json:"device"`
	Model    string `json:"model"`
	Channel  string `json:"channel"` // "red", "green", or "blue"
}

// Validate checks that all required fields are set.
func (cfg *LightColorConfig) Validate(_ string) ([]string, []string, error) {
	if cfg.APIKey == "" {
		return nil, nil, fmt.Errorf("api_key is required")
	}
	if cfg.DeviceID == "" {
		return nil, nil, fmt.Errorf("device (MAC address) is required")
	}
	if cfg.Model == "" {
		return nil, nil, fmt.Errorf("model is required")
	}
	switch cfg.Channel {
	case "red", "green", "blue":
	default:
		return nil, nil, fmt.Errorf("channel must be \"red\", \"green\", or \"blue\", got %q", cfg.Channel)
	}
	return nil, nil, nil
}

type goveeLightColor struct {
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	name   resource.Name
	logger logging.Logger
	cfg    *LightColorConfig

	client *govee.Client
}

func newGoveeLightColor(ctx context.Context, _ resource.Dependencies, rawConf resource.Config, logger logging.Logger) (toggleswitch.Switch, error) {
	conf, err := resource.NativeConfig[*LightColorConfig](rawConf)
	if err != nil {
		return nil, err
	}

	client, _, err := connectToDevice(ctx, conf.APIKey, conf.DeviceID, conf.Model, logger)
	if err != nil {
		return nil, err
	}

	return &goveeLightColor{
		name:   rawConf.ResourceName(),
		logger: logger,
		cfg:    conf,
		client: client,
	}, nil
}

func (s *goveeLightColor) Name() resource.Name {
	return s.name
}

func (s *goveeLightColor) DoCommand(_ context.Context, _ map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

// SetPosition sets the configured RGB channel to the given value (0-255).
// If all channels become 0 the light is turned off. Otherwise the light is
// turned on and the full RGB color is sent to the device.
func (s *goveeLightColor) SetPosition(ctx context.Context, position uint32, _ map[string]interface{}) error {
	if position > 255 {
		return fmt.Errorf("position must be 0-255, got %d", position)
	}

	// Read current color to preserve the other two channels.
	state, err := s.client.GetDeviceState(ctx, s.cfg.DeviceID, s.cfg.Model)
	if err != nil {
		return fmt.Errorf("failed to get device state: %w", err)
	}

	r, g, b := 0, 0, 0
	if state.Color != nil {
		r, g, b = state.Color.R, state.Color.G, state.Color.B
	}

	channelValue := int(position)
	switch s.cfg.Channel {
	case "red":
		r = channelValue
	case "green":
		g = channelValue
	case "blue":
		b = channelValue
	}

	if r == 0 && g == 0 && b == 0 {
		return s.client.TurnOff(ctx, s.cfg.DeviceID, s.cfg.Model)
	}

	if err := s.client.TurnOn(ctx, s.cfg.DeviceID, s.cfg.Model); err != nil {
		return err
	}
	return s.client.SetColor(ctx, s.cfg.DeviceID, s.cfg.Model, r, g, b)
}

// GetPosition returns the current value of the configured RGB channel (0-255).
func (s *goveeLightColor) GetPosition(ctx context.Context, _ map[string]interface{}) (uint32, error) {
	state, err := s.client.GetDeviceState(ctx, s.cfg.DeviceID, s.cfg.Model)
	if err != nil {
		return 0, fmt.Errorf("failed to get device state: %w", err)
	}

	if state.PowerState == "off" || state.Color == nil {
		return 0, nil
	}

	switch s.cfg.Channel {
	case "red":
		return uint32(state.Color.R), nil
	case "green":
		return uint32(state.Color.G), nil
	case "blue":
		return uint32(state.Color.B), nil
	}

	return 0, fmt.Errorf("unknown channel %q", s.cfg.Channel)
}

func (s *goveeLightColor) GetNumberOfPositions(_ context.Context, _ map[string]interface{}) (uint32, []string, error) {
	return 256, nil, nil
}
