package viamgovee

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	govee "github.com/DTCurrie/govee-go"
	"go.viam.com/rdk/components/sensor"
	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
	"go.viam.com/rdk/utils"
)

// GoveeDiscovery is the model identifier for the govee-discovery service.
var GoveeDiscovery = family.WithModel("govee-discovery")

func init() {
	resource.RegisterService(discovery.API, GoveeDiscovery,
		resource.Registration[discovery.Service, *DiscoveryConfig]{
			Constructor: newGoveeDiscover,
		},
	)
}

// DiscoveryConfig is the configuration for the govee-discovery service.
type DiscoveryConfig struct {
	APIKey string `json:"api_key"`
}

// Validate checks that the required api_key field is set.
func (cfg *DiscoveryConfig) Validate(_ string) ([]string, []string, error) {
	if cfg.APIKey == "" {
		return nil, nil, fmt.Errorf("api_key is required")
	}
	return nil, nil, nil
}

// GoveeDiscover is the discovery service that lists all Govee devices and
// emits Viam resource configs for each one.
type GoveeDiscover struct {
	resource.AlwaysRebuild
	resource.TriviallyCloseable

	name   resource.Name
	logger logging.Logger
	cfg    *DiscoveryConfig
	client *govee.Client
}

func newGoveeDiscover(ctx context.Context, _ resource.Dependencies, rawConf resource.Config, logger logging.Logger) (discovery.Service, error) {
	conf, err := resource.NativeConfig[*DiscoveryConfig](rawConf)
	if err != nil {
		return nil, err
	}

	client := govee.New(conf.APIKey)

	// Verify credentials by listing devices on startup.
	if _, err := client.GetDevices(ctx); err != nil {
		return nil, fmt.Errorf("govee: cannot connect with provided api_key: %w", err)
	}

	return &GoveeDiscover{
		name:   rawConf.ResourceName(),
		logger: logger,
		cfg:    conf,
		client: client,
	}, nil
}

// Name returns the resource name of the discovery service.
func (s *GoveeDiscover) Name() resource.Name {
	return s.name
}

// DoCommand is a no-op implementation of the resource.Resource interface.
func (s *GoveeDiscover) DoCommand(_ context.Context, _ map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

// DiscoverResources returns Viam resource configs for all discovered Govee devices.
func (s *GoveeDiscover) DiscoverResources(ctx context.Context, _ map[string]any) ([]resource.Config, error) {
	return s.discoverGovee(ctx)
}

// reUnsafe replaces any character that is not alphanumeric, '-', or '_' with '-'.
var reUnsafe = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// reCollapse collapses consecutive '-' characters into one.
var reCollapse = regexp.MustCompile(`-{2,}`)

// sanitizeName produces a safe Viam resource name from a device name.
func sanitizeName(name string) string {
	s := reUnsafe.ReplaceAllString(name, "-")
	s = reCollapse.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func (s *GoveeDiscover) discoverGovee(ctx context.Context) ([]resource.Config, error) {
	devices, err := s.client.GetDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("govee: cannot list devices: %w", err)
	}

	var configs []resource.Config
	var colorDevices []govee.Device

	for _, device := range devices {
		supportsColor := device.SupportsCmd("color")

		s.logger.Debugf("discovered govee device: %s (%s) model=%s controllable=%v retrievable=%v",
			device.DeviceName, device.DeviceID, device.Model, device.Controllable, device.Retrievable)

		safeName := sanitizeName(device.DeviceName)

		baseAttrs := utils.AttributeMap{
			"api_key": s.cfg.APIKey,
			"device":  device.DeviceID,
			"model":   device.Model,
		}

		// All devices with brightness support get a brightness switch.
		if device.SupportsCmd("brightness") || device.SupportsCmd("turn") {
			configs = append(configs, resource.Config{
				Name:       safeName,
				API:        toggleswitch.API,
				Model:      GoveeLightBrightness,
				Attributes: baseAttrs,
			})
		}

		// All retrievable devices get a sensor.
		if device.Retrievable {
			configs = append(configs, resource.Config{
				Name:       fmt.Sprintf("%s-sensor", safeName),
				API:        sensor.API,
				Model:      GoveeLightSensor,
				Attributes: baseAttrs,
			})
		}

		// Color-capable devices get one switch per RGB channel.
		if supportsColor {
			colorDevices = append(colorDevices, device)
			for _, channel := range []string{"red", "green", "blue"} {
				channelAttrs := utils.AttributeMap{
					"api_key": s.cfg.APIKey,
					"device":  device.DeviceID,
					"model":   device.Model,
					"channel": channel,
				}
				configs = append(configs, resource.Config{
					Name:       fmt.Sprintf("%s-%s", safeName, channel),
					API:        toggleswitch.API,
					Model:      GoveeLightColor,
					Attributes: channelAttrs,
				})
			}
		}
	}

	// Emit a single mode switch covering all color-capable devices.
	if len(colorDevices) > 0 {
		modeDevices := make([]map[string]interface{}, 0, len(colorDevices))
		for _, d := range colorDevices {
			modeDevices = append(modeDevices, map[string]interface{}{
				"device": d.DeviceID,
				"model":  d.Model,
			})
		}
		configs = append(configs, resource.Config{
			Name:  "govee-mode",
			API:   toggleswitch.API,
			Model: GoveeLightMode,
			Attributes: utils.AttributeMap{
				"api_key":  s.cfg.APIKey,
				"daylight": modeDevices,
				"warm":     modeDevices,
			},
		})
	}

	return configs, nil
}
