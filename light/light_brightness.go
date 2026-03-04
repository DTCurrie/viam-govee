package light

import (
	"context"
	"fmt"

	govee "github.com/DTCurrie/govee-go"
	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"

	"github.com/DTCurrie/viam-govee/internal/common"
)

// GoveeLightBrightness is the model identifier for the govee-light-brightness component.
var GoveeLightBrightness = common.Family.WithModel("govee-light-brightness")

func init() {
	resource.RegisterComponent(toggleswitch.API, GoveeLightBrightness,
		resource.Registration[toggleswitch.Switch, *BrightnessConfig]{
			Constructor: newGoveeLightBrightness,
		},
	)
}

// BrightnessConfig is the configuration for the govee-light-brightness component.
type BrightnessConfig struct {
	APIKey   string `json:"api_key"`
	DeviceID string `json:"device"`
	SKU      string `json:"sku"`
}

// Validate checks that all required fields are set.
func (cfg *BrightnessConfig) Validate(_ string) ([]string, []string, error) {
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

type goveeLightBrightness struct {
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	name   resource.Name
	logger logging.Logger
	cfg    *BrightnessConfig

	client *govee.Client
}

func newGoveeLightBrightness(ctx context.Context, _ resource.Dependencies, rawConf resource.Config, logger logging.Logger) (toggleswitch.Switch, error) {
	conf, err := resource.NativeConfig[*BrightnessConfig](rawConf)
	if err != nil {
		return nil, err
	}

	client, _, err := common.ConnectToDevice(ctx, conf.APIKey, conf.DeviceID, conf.SKU, logger)
	if err != nil {
		return nil, err
	}

	return &goveeLightBrightness{
		name:   rawConf.ResourceName(),
		logger: logger,
		cfg:    conf,
		client: client,
	}, nil
}

func (s *goveeLightBrightness) Name() resource.Name {
	return s.name
}

func (s *goveeLightBrightness) DoCommand(_ context.Context, _ map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

// SetPosition sets the brightness of the light.
// Positions 1-100 map 1:1 to API brightness 1-100%.
func (s *goveeLightBrightness) SetPosition(ctx context.Context, position uint32, _ map[string]interface{}) error {
	if position < 1 || position > 100 {
		return fmt.Errorf("position must be 1-100, got %d", position)
	}
	return s.client.SetBrightness(ctx, s.cfg.SKU, s.cfg.DeviceID, int(position))
}

// GetPosition returns the current brightness as a switch position (1-100),
// or 0 if the brightness is unavailable.
func (s *goveeLightBrightness) GetPosition(ctx context.Context, _ map[string]interface{}) (uint32, error) {
	state, err := s.client.GetDeviceState(ctx, s.cfg.SKU, s.cfg.DeviceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get device state: %w", err)
	}

	bri, ok := state.Brightness()
	if !ok || bri <= 0 {
		return 0, nil
	}
	return uint32(bri), nil
}

func (s *goveeLightBrightness) GetNumberOfPositions(_ context.Context, _ map[string]interface{}) (uint32, []string, error) {
	return 100, nil, nil
}
