package discover

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

	"github.com/DTCurrie/viam-govee/internal/common"
	"github.com/DTCurrie/viam-govee/light"
	"github.com/DTCurrie/viam-govee/plug"
)

// GoveeDiscovery is the model identifier for the govee-discovery service.
var GoveeDiscovery = common.Family.WithModel("govee-discovery")

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

	for _, device := range devices {
		isPlug := device.Type == govee.DeviceSocket
		supportsColor := device.HasCapability(govee.CapabilityColorSetting, "colorRgb")
		supportsColorTemp := device.HasCapability(govee.CapabilityColorSetting, "colorTemperatureK")
		supportsBrightness := device.HasCapability(govee.CapabilityRange, "brightness")
		supportsScene := device.HasCapability(govee.CapabilityDynamicScene, "lightScene")
		supportsDIYScene := device.HasCapability(govee.CapabilityDynamicScene, "diyScene")
		supportsSnapshot := device.HasCapability(govee.CapabilityDynamicScene, "snapshot")

		supportsOnOff := device.HasCapability(govee.CapabilityOnOff, "powerSwitch")

		s.logger.Debugf("discovered govee device: %s (%s) sku=%s type=%s plug=%v on_off=%v color=%v color_temp=%v brightness=%v scene=%v diy_scene=%v snapshot=%v",
			device.DeviceName, device.DeviceID, device.SKU, device.Type, isPlug, supportsOnOff, supportsColor, supportsColorTemp, supportsBrightness, supportsScene, supportsDIYScene, supportsSnapshot)

		safeName := sanitizeName(device.DeviceName)

		baseAttrs := utils.AttributeMap{
			"api_key": s.cfg.APIKey,
			"device":  device.DeviceID,
			"sku":     device.SKU,
		}

		if isPlug {
			configs = append(configs, resource.Config{
				Name:       safeName,
				API:        toggleswitch.API,
				Model:      plug.GoveePlugSwitch,
				Attributes: baseAttrs,
			})
			configs = append(configs, resource.Config{
				Name:       fmt.Sprintf("%s-sensor", safeName),
				API:        sensor.API,
				Model:      plug.GoveePlugSensor,
				Attributes: baseAttrs,
			})
			continue
		}

		// On/off switch for any light that supports on/off.
		if supportsOnOff {
			configs = append(configs, resource.Config{
				Name:       fmt.Sprintf("%s-switch", safeName),
				API:        toggleswitch.API,
				Model:      light.GoveeLightSwitch,
				Attributes: baseAttrs,
			})
		}

		// Brightness switch for any light that supports brightness control.
		if supportsBrightness {
			configs = append(configs, resource.Config{
				Name:       safeName,
				API:        toggleswitch.API,
				Model:      light.GoveeLightBrightness,
				Attributes: baseAttrs,
			})
		}

		// Sensor for every light device.
		configs = append(configs, resource.Config{
			Name:       fmt.Sprintf("%s-sensor", safeName),
			API:        sensor.API,
			Model:      light.GoveeLightSensor,
			Attributes: baseAttrs,
		})

		// Color temperature switch for lights that support color temperature.
		if supportsColorTemp {
			configs = append(configs, resource.Config{
				Name:       fmt.Sprintf("%s-color-temp", safeName),
				API:        toggleswitch.API,
				Model:      light.GoveeLightColorTemp,
				Attributes: baseAttrs,
			})
		}

		// Per-channel color switches for color-capable lights.
		if supportsColor {
			for _, channel := range []string{"red", "green", "blue"} {
				channelAttrs := utils.AttributeMap{
					"api_key": s.cfg.APIKey,
					"device":  device.DeviceID,
					"sku":     device.SKU,
					"channel": channel,
				}
				configs = append(configs, resource.Config{
					Name:       fmt.Sprintf("%s-%s", safeName, channel),
					API:        toggleswitch.API,
					Model:      light.GoveeLightColor,
					Attributes: channelAttrs,
				})
			}
		}

		// Scene switch for lights that support dynamic scenes.
		if supportsScene {
			configs = append(configs, resource.Config{
				Name:       fmt.Sprintf("%s-scene", safeName),
				API:        toggleswitch.API,
				Model:      light.GoveeLightScene,
				Attributes: baseAttrs,
			})
		}

		// DIY scene switch for lights that support user-created DIY scenes.
		if supportsDIYScene {
			configs = append(configs, resource.Config{
				Name:       fmt.Sprintf("%s-diy-scene", safeName),
				API:        toggleswitch.API,
				Model:      light.GoveeLightDIYScene,
				Attributes: baseAttrs,
			})
		}

		// Snapshot switch for lights that support saved snapshots.
		if supportsSnapshot {
			configs = append(configs, resource.Config{
				Name:       fmt.Sprintf("%s-snapshot", safeName),
				API:        toggleswitch.API,
				Model:      light.GoveeLightSnapshot,
				Attributes: baseAttrs,
			})
		}
	}

	return configs, nil
}
