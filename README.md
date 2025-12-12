# 🔁 PID 

`pid` provides a configurable [PID controller](https://en.wikipedia.org/wiki/Proportional%E2%80%93integral%E2%80%93derivative_controller) in Go for feedback-driven systems. While PID controllers are commonly used for physical control, the same feedback principles can apply to backend and infrastructure systems.

Typical use cases include smoothing load, regulating rates, or adapting resource usage based on observed metrics, where gradual adjustment is preferable to fixed thresholds.

## Installation

```bash
go get github.com/konradreiche/pid@latest
```

## Usage

The example below demonstrates a basic control loop. Additional configuration options and runnable examples are available in the package documentation.

```go
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
```

## Acknowledgements

A special thanks to Donovan Baarda for discussions and insights from our collaboration on the [DLFU Cache](https://minkirri.apana.org.au/wiki/DecayingLFUCacheExpiry), including implementation details that helped clarify feedback and control principles used in this library, as well as his perspectives on why [thresholds and limits are problematic](https://minkirri.apana.org.au/wiki/ThresholdsAndLimitsAreBad).
