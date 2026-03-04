package main

import (
	"go.viam.com/rdk/components/sensor"
	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"

	"github.com/DTCurrie/viam-govee/discover"
	"github.com/DTCurrie/viam-govee/light"
	"github.com/DTCurrie/viam-govee/plug"
)

func main() {
	module.ModularMain(
		resource.APIModel{API: toggleswitch.API, Model: light.GoveeLightSwitch},
		resource.APIModel{API: toggleswitch.API, Model: light.GoveeLightBrightness},
		resource.APIModel{API: toggleswitch.API, Model: light.GoveeLightColor},
		resource.APIModel{API: toggleswitch.API, Model: light.GoveeLightColorTemp},
		resource.APIModel{API: toggleswitch.API, Model: light.GoveeLightScene},
		resource.APIModel{API: toggleswitch.API, Model: light.GoveeLightDIYScene},
		resource.APIModel{API: toggleswitch.API, Model: light.GoveeLightSnapshot},
		resource.APIModel{API: toggleswitch.API, Model: plug.GoveePlugSwitch},
		resource.APIModel{API: discovery.API, Model: discover.GoveeDiscovery},
		resource.APIModel{API: sensor.API, Model: light.GoveeLightSensor},
		resource.APIModel{API: sensor.API, Model: plug.GoveePlugSensor},
	)
}
