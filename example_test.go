package pid_test

import (
	"fmt"
	"log"
	"time"

	"github.com/konradreiche/pid"
)

func ExampleNew() {
	controller, err := pid.New(
		pid.WithStandardForm(2.0, 1.0, 0.25),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%#v\n", controller)
	// Output: &pid.Controller{proportionalGain:2, integralGain:2, derivativeGain:0.5, prevControlError:0, integral:0, derivative:0, outputLimit:pid.limit{lower:-Inf, upper:+Inf}, integralLimit:pid.limit{lower:-Inf, upper:+Inf}, lowPassFilterError:0.00390625, lowPassFilterDerivative:0.03125, trapezoidalIntegral:false, metrics:(*pid.metrics)(nil)}
}

func ExampleController_Update() {
	controller, err := pid.New(
		pid.WithStandardForm(1.5, 1.0, 0.2),
		pid.WithTrapezoidalIntegral(true),
		pid.WithOutputLimit(-1, 1),
	)
	if err != nil {
		log.Fatal(err)
	}

	const target = 1.0
	timeStep := 100 * time.Millisecond

	// The control signal starts clamped at the output limit, then tapers off as
	// the measurement converges on the target.
	var measurement float64
	for i := range 5 {
		control := controller.Update(target, measurement, timeStep)
		measurement += 0.25 * control
		fmt.Printf("step=%d control=%.2f measurement=%.2f\n", i, control, measurement)
	}

	// Output:
	// step=0 control=1.00 measurement=0.25
	// step=1 control=1.00 measurement=0.50
	// step=2 control=0.45 measurement=0.61
	// step=3 control=0.55 measurement=0.75
	// step=4 control=0.39 measurement=0.85
}
