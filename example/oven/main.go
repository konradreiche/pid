package main

import (
	"log"
	"net/http"
	"time"

	"github.com/konradreiche/pid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	targetTemperature        = 350.0
	ambientTemperature       = 70
	maxHeaterPower           = 20.0
	temperatureLossPerSecond = 0.01
)

var (
	ovenTemperature = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "oven_temperature",
	}, []string{"controller"})

	ovenTargetTemperature = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "oven_target_temperature",
	}, []string{"controller"})

	ovenPowerRatio = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "oven_power_ratio",
	}, []string{"controller"})
)

// This demo runs two controllers against the same simplified oven model:

// 1. On-Off Controller (Thermostat with Deadband)
// 2. PID Controller (Tuned via Ziegler-Nichols)
//
// The goal is not physical accuracy, but to produce repeatable dynamics (heat
// input, proportional heat loss, and a disturbance event) that make controller
// behavior easy to visualize and observe via Prometheus metrics.
func main() {
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Metrics available at http://localhost:2112/metrics (Prometheus on :9090)")

	runOnOffController()
	runPIDController()

	if err := http.ListenAndServe(":2112", nil); err != nil {
		log.Fatal(err)
	}
}

func runOnOffController() {
	onOffControllerSimulation := newSimulation("on-off")
	go onOffControllerSimulation.run(newOnOffController())
}

func runPIDController() {
	pidSimulation := newSimulation("pid")

	// Example configuration that will produce oscillations:
	//	pid, err := pid.New(
	//		pid.WithIntegralGain(0.0),
	//		pid.WithDerivativeGain(0.0),
	//		pid.WithProportionalGain(2.0),
	//		pid.WithOutputLimit(0, 20),
	//		pid.WithPrometheusMetrics("demo", prometheus.DefaultRegisterer),
	//	)

	pid, err := pid.New(
		pid.WithZieglerNicholsMethod(1.7, 2),
		pid.WithTrapezoidalIntegral(true),
		pid.WithOutputLimit(0.0, 20.0),
		pid.WithPrometheusMetrics("demo", prometheus.DefaultRegisterer),
	)
	if err != nil {
		log.Fatal(err)
	}
	go pidSimulation.run(pid)
}

func newSimulation(name string) *simulation {
	oven := &oven{
		currentTemperature:       ambientTemperature,
		temperatureLossPerSecond: temperatureLossPerSecond,
		targetTemperature:        targetTemperature,
		maxHeatRate:              maxHeaterPower,
	}
	simulation := &simulation{
		name:     name,
		oven:     oven,
		timeStep: 1 * time.Second,
	}
	return simulation
}

type controller interface {
	// Update returns the controller output, interpreted here as heater power and
	// should return a result in the range of  [0, maxHeaterPower].
	Update(target, current float64, delta time.Duration) float64
}

// onOffController implements a simple thermostat-style controller with a
// deadband. Deadband is a tolerance zone around the target temperature. Within
// this range the heater stays off, preventing rapid on/off cycling.
type onOffController struct{}

func newOnOffController() *onOffController {
	return &onOffController{}
}

func (o *onOffController) Update(target, current float64, delta time.Duration) float64 {
	const deadband = 5
	if current < target-deadband {
		return maxHeaterPower
	}
	return 0.0
}

// oven is a deliberately simplified first-order thermal model.
//
// - Heating: temperature rises linearly with applied power.
// - Cooling: temperature loss is proportional to (current - ambient).
//
// These assumptions create a stable baseline that highlights controller
// behavior (overshoot, settling time, recovery from disturbance) without
// complex math.
type oven struct {
	currentTemperature       float64
	temperatureLossPerSecond float64
	targetTemperature        float64
	maxHeatRate              float64
}

type simulation struct {
	name     string
	oven     *oven
	timeStep time.Duration
}

func (s *simulation) run(controller controller) {
	var i int
	for {
		ovenTemperature.WithLabelValues(s.name).Set(s.oven.currentTemperature)
		ovenTargetTemperature.WithLabelValues(s.name).Set(s.oven.targetTemperature)

		control := controller.Update(
			s.oven.targetTemperature,
			s.oven.currentTemperature,
			s.timeStep,
		)

		// powerRatio is the fraction of maximum oven heater power currently
		// applied.
		powerRatio := control / maxHeaterPower
		ovenPowerRatio.WithLabelValues(s.name).Set(powerRatio)

		// Heating adds temperature linearly with power.
		if powerRatio > 0 {
			s.oven.currentTemperature += s.oven.maxHeatRate * powerRatio * s.timeStep.Seconds()
		}

		// Cooling removes heat faster as the oven gets hotter than ambient.
		if delta := s.oven.currentTemperature - ambientTemperature; delta > 0 {
			s.oven.currentTemperature -= s.oven.temperatureLossPerSecond * delta * s.timeStep.Seconds()
		}

		// Disturbance: a one-time step drop simulates opening the door. This is useful
		// to compare how controllers recover from an exogenous shock.
		if i == 43 {
			s.oven.currentTemperature -= 60
		}

		if i%10 == 0 {
			log.Printf(
				"%-6s temperature = %5.1f°F target = %3.0f°F power = %4.2f",
				s.name,
				s.oven.currentTemperature,
				s.oven.targetTemperature,
				powerRatio,
			)
		}

		// Sleep to pace the simulation in real time so Prometheus scraping and
		// graphs reflect progression naturally.
		time.Sleep(s.timeStep)
		i++
	}
}
