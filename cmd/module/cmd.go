package main

import (
	"go.viam.com/rdk/components/sensor"
	toggleswitch "go.viam.com/rdk/components/switch"
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"

	viamgovee "github.com/DTCurrie/viam-govee"
)

func main() {
	module.ModularMain(
		resource.APIModel{API: toggleswitch.API, Model: viamgovee.GoveeLightBrightness},
		resource.APIModel{API: toggleswitch.API, Model: viamgovee.GoveeLightColor},
		resource.APIModel{API: toggleswitch.API, Model: viamgovee.GoveeLightMode},
		resource.APIModel{API: discovery.API, Model: viamgovee.GoveeDiscovery},
		resource.APIModel{API: sensor.API, Model: viamgovee.GoveeLightSensor},
	)
}
