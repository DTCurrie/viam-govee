package light

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	govee "github.com/DTCurrie/govee-go"
	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"

	"github.com/DTCurrie/viam-govee/internal/common"
)

// GoveeLightColorTemp is the model identifier for the govee-light-color-temp component.
var GoveeLightColorTemp = common.Family.WithModel("govee-light-color-temp")

func init() {
	resource.RegisterComponent(toggleswitch.API, GoveeLightColorTemp,
		resource.Registration[toggleswitch.Switch, *ColorTempConfig]{
			Constructor: newGoveeLightColorTemp,
		},
	)
}

// ColorTempConfig is the configuration for the govee-light-color-temp component.
type ColorTempConfig struct {
	APIKey   string `json:"api_key"`
	DeviceID string `json:"device"`
	SKU      string `json:"sku"`
}

// Validate checks that all required fields are set.
func (cfg *ColorTempConfig) Validate(_ string) ([]string, []string, error) {
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

type goveeLightColorTemp struct {
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	name   resource.Name
	logger logging.Logger
	cfg    *ColorTempConfig

	client *govee.Client
	minK   int // minimum color temperature in Kelvin supported by device
	maxK   int // maximum color temperature in Kelvin supported by device
}

// colorTempRange holds the integer range parameters for the colorTemperatureK capability.
type colorTempRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// colorTempParams holds the parameters object of the colorTemperatureK capability.
type colorTempParams struct {
	Range colorTempRange `json:"range"`
}

func newGoveeLightColorTemp(ctx context.Context, _ resource.Dependencies, rawConf resource.Config, logger logging.Logger) (toggleswitch.Switch, error) {
	conf, err := resource.NativeConfig[*ColorTempConfig](rawConf)
	if err != nil {
		return nil, err
	}

	client, device, err := common.ConnectToDevice(ctx, conf.APIKey, conf.DeviceID, conf.SKU, logger)
	if err != nil {
		return nil, err
	}

	minK, maxK, err := parseColorTempRange(device)
	if err != nil {
		return nil, fmt.Errorf("govee: device %s does not support color temperature: %w", conf.DeviceID, err)
	}

	return &goveeLightColorTemp{
		name:   rawConf.ResourceName(),
		logger: logger,
		cfg:    conf,
		client: client,
		minK:   minK,
		maxK:   maxK,
	}, nil
}

// parseColorTempRange extracts the Kelvin min/max range from the device's
// colorTemperatureK capability parameters.
func parseColorTempRange(device *govee.Device) (minK, maxK int, err error) {
	cap := device.FindCapability(govee.CapabilityColorSetting, "colorTemperatureK")
	if cap == nil {
		return 0, 0, fmt.Errorf("colorTemperatureK capability not found")
	}
	if cap.Parameters == nil {
		return 0, 0, fmt.Errorf("colorTemperatureK capability has no parameters")
	}

	var params colorTempParams
	if err := json.Unmarshal(cap.Parameters, &params); err != nil {
		return 0, 0, fmt.Errorf("failed to parse colorTemperatureK parameters: %w", err)
	}

	if params.Range.Min <= 0 || params.Range.Max <= params.Range.Min {
		return 0, 0, fmt.Errorf("invalid colorTemperatureK range [%d, %d]", params.Range.Min, params.Range.Max)
	}

	return params.Range.Min, params.Range.Max, nil
}

func (s *goveeLightColorTemp) Name() resource.Name {
	return s.name
}

func (s *goveeLightColorTemp) DoCommand(_ context.Context, _ map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

// SetPosition sets the color temperature of the light.
// Positions 1-100 map linearly from the device's minimum to maximum Kelvin range:
// position 1 = minK (warmest), position 100 = maxK (coolest).
func (s *goveeLightColorTemp) SetPosition(ctx context.Context, position uint32, _ map[string]interface{}) error {
	if position < 1 || position > 100 {
		return fmt.Errorf("position must be 1-100, got %d", position)
	}

	kelvin := s.minK + int(math.Round(float64(position-1)/99.0*float64(s.maxK-s.minK)))
	return s.client.SetColorTemp(ctx, s.cfg.SKU, s.cfg.DeviceID, kelvin)
}

// GetPosition returns the current color temperature mapped back to a switch position (1-100),
// or 0 if the color temperature is unavailable.
func (s *goveeLightColorTemp) GetPosition(ctx context.Context, _ map[string]interface{}) (uint32, error) {
	state, err := s.client.GetDeviceState(ctx, s.cfg.SKU, s.cfg.DeviceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get device state: %w", err)
	}

	kelvin, ok := state.ColorTemp()
	if !ok || kelvin == 0 {
		return 0, nil
	}

	if kelvin <= s.minK {
		return 1, nil
	}
	if kelvin >= s.maxK {
		return 100, nil
	}

	pos := 1 + uint32(math.Round(float64(kelvin-s.minK)/float64(s.maxK-s.minK)*99.0))
	return pos, nil
}

func (s *goveeLightColorTemp) GetNumberOfPositions(_ context.Context, _ map[string]interface{}) (uint32, []string, error) {
	return 100, nil, nil
}
