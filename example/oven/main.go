package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/konradreiche/pid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	targetTemperature        = 300.0
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

var (
	csv     = flag.Bool("csv", false, "write simulation data to CSV files for each controller")
	seconds = flag.Int("seconds", 120, "number of seconds to run in CSV mode")
)

// This demo runs two controllers against the same simplified oven model:

// 1. On-Off Controller (thermostat with deadband)
// 2. PID Controller (proportional only)
// 3. PID Controller at ultimate gain (sustained oscillation)
// 4. PID Controller tuned using the Ziegler–Nichols method
//
// The goal is not physical accuracy, but to produce repeatable dynamics (heat
// input and proportional heat loss) that make controller behavior easy to
// visualize and observe via Prometheus metrics.
func main() {
	flag.Parse()

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Metrics available at http://localhost:2112/metrics (Prometheus on :9090)")

	runOnOffController()

	runPIDController("p_only",
		pid.WithProportionalGain(1.0),
		pid.WithIntegralGain(0.0),
		pid.WithDerivativeGain(0.0),
		pid.WithOutputLimit(0.0, maxHeaterPower),
	)

	runPIDController("pi",
		pid.WithProportionalGain(1.0),
		pid.WithIntegralGain(3.0),
		pid.WithDerivativeGain(0.0),
		pid.WithOutputLimit(0.0, maxHeaterPower),
	)

	runPIDController("pid",
		pid.WithProportionalGain(1.0),
		pid.WithIntegralGain(3.0),
		pid.WithDerivativeGain(0.5),
		pid.WithOutputLimit(0.0, maxHeaterPower),
	)

	runPIDController("ultimate_gain",
		pid.WithProportionalGain(2.01),
		pid.WithIntegralGain(0.0),
		pid.WithDerivativeGain(0.0),
		pid.WithOutputLimit(0, maxHeaterPower),
	)

	runPIDController("zn_pid",
		pid.WithZieglerNicholsMethod(2.01, 2),
		pid.WithOutputLimit(0.0, maxHeaterPower),
	)

	runPIDController("zn_pi",
		pid.WithZieglerNicholsMethod(2.01, 2),
		pid.WithDerivativeGain(0.0),
		pid.WithOutputLimit(0.0, maxHeaterPower),
	)

	runPIDController("zn_pi_anti_windup",
		pid.WithZieglerNicholsMethod(2.01, 2),
		pid.WithDerivativeGain(0.0),
		pid.WithOutputLimit(0.0, maxHeaterPower),
		pid.WithAntiWindupClamping(true),
	)

	runPIDController("zn_pi_setpoint_filter",
		pid.WithZieglerNicholsMethod(2.01, 2),
		pid.WithDerivativeGain(0.0),
		pid.WithOutputLimit(0.0, maxHeaterPower),
		pid.WithSetpointFilter(4),
	)

	if err := http.ListenAndServe(":2112", nil); err != nil {
		log.Fatal(err)
	}
}

func runOnOffController() {
	onOffControllerSimulation := newSimulation("on-off")
	go onOffControllerSimulation.run(newOnOffController())
}

func runPIDController(name string, opts ...pid.Option) {
	pidSimulation := newSimulation(name)
	pid, err := pid.New(
		pid.WithOptions(opts...),
		pid.WithPrometheusMetrics(name, prometheus.DefaultRegisterer),
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
	if *csv {
		simulation.csvFile = createCSV(name)
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

func (c *onOffController) Update(target, current float64, delta time.Duration) float64 {
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
	csvFile  *os.File
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
		s.writeToCSVFile(i)

		// Heating adds temperature linearly with power.
		if powerRatio > 0 {
			s.oven.currentTemperature += s.oven.maxHeatRate * powerRatio * s.timeStep.Seconds()
		}

		// Cooling removes heat faster as the oven gets hotter than ambient.
		if delta := s.oven.currentTemperature - ambientTemperature; delta > 0 {
			s.oven.currentTemperature -= s.oven.temperatureLossPerSecond * delta * s.timeStep.Seconds()
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
		if !*csv {
			time.Sleep(s.timeStep)
		}
		if *csv && i == *seconds {
			break
		}
		i++
	}
}

func (s *simulation) writeToCSVFile(step int) {
	if !*csv {
		return
	}
	if _, err := fmt.Fprintf(
		s.csvFile,
		"%d,%.2f\n",
		step,
		s.oven.currentTemperature,
	); err != nil {
		log.Fatal(err)
	}
	if err := s.csvFile.Sync(); err != nil {
		log.Fatal(err)
	}
}

func createCSV(name string) *os.File {
	outputDir := "output"
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		log.Fatal(err)
	}
	csvFile, err := os.Create(filepath.Join(outputDir, name+".csv"))
	if err != nil {
		log.Fatal(err)
	}
	return csvFile
}
