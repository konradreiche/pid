// Package pid provides a configurable PID controller for feedback-based
// control systems.
package pid

import (
	"math"
	"time"
)

// Controller implements a PID controller for feedback-driven systems, commonly
// used to regulate physical processes such as temperature control but the same
// feedback principles can apply to software systems. In backend and
// infrastructure contexts, a Controller can be used to smooth load, regulate
// rates, or adapt resource usage based on observed metrics rather than fixed
// thresholds.
type Controller struct {
	// Proportional Gain (𝐾𝑝) controls the response to the current error, higher proportionalGain
	// reduces the rise time but can cause overshoot and oscillations.
	proportionalGain float64
	// Integral gain (𝐾𝑖) addresses accumulated past errors (offsets), higher integralGain
	// eliminates steady-state error but may cause instability or slow response.
	integralGain float64
	// Derivative gain (𝐾𝑑) predicts future error based on its rate of change, higher
	// derivativeGain reduces overshoot and oscillations but can amplify noise.
	derivativeGain float64

	prevControlError float64
	integral         float64
	derivative       float64

	// Limits ensure that the controller operates within safe bounds and to
	// prevent integral windup (overshoot, slow recovery, oscillation).
	outputLimit   limit
	integralLimit limit

	lowPassFilterError      float64
	lowPassFilterDerivative float64
	trapezoidalIntegral     bool
}

// New constructs a [*Controller] configured by the provided options.
// Reasonable defaults are used when options are omitted.
func New(opts ...Option) (*Controller, error) {
	cfg := options{
		proportionalGain: 1.0,
		integralGain:     0.0,
		derivativeGain:   0.0,
		outputLimit:      newLimit(math.Inf(-1), math.Inf(1)),
	}
	if err := WithOptions(opts...)(&cfg); err != nil {
		return nil, err
	}

	integralLimit := newLimit(math.Inf(-1), math.Inf(1))
	if cfg.integralGain > 0.0 {
		integralLimit = newLimit(
			cfg.outputLimit.lower/cfg.integralGain,
			cfg.outputLimit.upper/cfg.integralGain,
		)
	}

	return &Controller{
		proportionalGain:        cfg.proportionalGain,
		integralGain:            cfg.integralGain,
		derivativeGain:          cfg.derivativeGain,
		outputLimit:             cfg.outputLimit,
		integralLimit:           integralLimit,
		trapezoidalIntegral:     cfg.trapezoidalIntegral,
		lowPassFilterError:      cfg.lowPassFilterError,
		lowPassFilterDerivative: cfg.lowPassFilterDerivative,
	}, nil
}

// Update computes and returns the next control signal for the given target and
// current measurement over the provided time step. Call Update once per
// control loop iteration, passing the time elapsed since the previous call.
func (c *Controller) Update(target, current float64, delta time.Duration) float64 {
	step := float64(delta.Seconds())

	// Calculate the error value as the difference between the target and current
	// value. This time-dependent error drives the PID terms (P, I, and D).
	controlError := target - current
	// Optionally apply a low-pass filter to reduce noise in the error signal.
	if c.lowPassFilterError != 0.0 {
		controlError = (controlError*step + c.prevControlError*c.lowPassFilterError) / (c.lowPassFilterError + step)
	}

	c.integral = c.integralLimit.apply(c.updateIntegral(controlError, step))
	c.derivative = c.updateDerivative(controlError, step)

	// Defer updating the previous control error until after computing the
	// integral and derivative, both depend on the prior error value.
	c.prevControlError = controlError

	output := c.proportionalGain*controlError + c.integralGain*c.integral + c.derivativeGain*c.derivative

	// Limits ensure that the controller operates within safe bounds and to
	// prevent integral windup (overshoot, slow recovery, oscillation).
	return c.outputLimit.apply(output)
}

// updateIntegral adds up past errors in every step to eliminate residual bias that
// the proportional and derivative terms can't fully correct.
func (c *Controller) updateIntegral(controlError, step float64) float64 {
	if c.trapezoidalIntegral {
		return c.integral + step*(controlError+c.prevControlError)/2.0
	}
	return c.integral + controlError*step
}

func (c *Controller) updateDerivative(controlError, step float64) float64 {
	derivative := (controlError - c.prevControlError) / step
	if c.lowPassFilterDerivative != 0.0 {
		derivative = ((controlError - c.prevControlError) + c.lowPassFilterDerivative*c.derivative) / (step + c.lowPassFilterDerivative)
	}
	return derivative
}

type options struct {
	proportionalGain        float64
	integralGain            float64
	derivativeGain          float64
	outputLimit             limit
	trapezoidalIntegral     bool
	lowPassFilterError      float64
	lowPassFilterDerivative float64
}

// Option is a functional option for flexible and extensible configuration of
// [*Controller], allowing modification of internal state or behavior during
// construction.
type Option func(*options) error

// WithZieglerNicholsMethod configures gains using the Ziegler-Nichols tuning
// method based on the supplied ultimate gain and oscillation period.
func WithZieglerNicholsMethod(
	ultimateGain float64,
	oscillationPeriod float64,
) Option {
	return WithStandardForm(
		0.6*ultimateGain,
		oscillationPeriod/2.0,
		oscillationPeriod/8.0,
	)
}

// WithStandardForm configures the controller using the standard PID form. The
// resulting integral and derivative gains are derived from these values.
func WithStandardForm(proportionalGain, integralTimeConstant, derivativeTimeConstant float64) Option {
	return func(o *options) error {
		o.proportionalGain = proportionalGain
		o.integralGain = proportionalGain / integralTimeConstant
		o.derivativeGain = proportionalGain * derivativeTimeConstant
		o.lowPassFilterDerivative = derivativeTimeConstant / 8.0
		o.lowPassFilterError = derivativeTimeConstant / 64.0
		return nil
	}
}

// WithProportionalGain sets the proportional gain (𝐾𝑝).
func WithProportionalGain(proportionalGain float64) Option {
	return func(o *options) error {
		o.proportionalGain = proportionalGain
		return nil
	}
}

// WithIntegralGain sets the integral gain (𝐾𝑖).
func WithIntegralGain(integralGain float64) Option {
	return func(o *options) error {
		o.integralGain = integralGain
		return nil
	}
}

// WithDerivativeGain sets the derivative gain (𝐾𝑑).
func WithDerivativeGain(derivativeGain float64) Option {
	return func(o *options) error {
		o.derivativeGain = derivativeGain
		return nil
	}
}

// WithLowPassFilterError includes a low-pass filter with the specified time
// constant which can be used to smooth out high-frequency changes. A large
// value results in a slow response and more smoothing. A small value results
// in a fast response and less smoothing.
func WithLowPassFilterError(lowPassFilterError float64) Option {
	return func(o *options) error {
		o.lowPassFilterError = lowPassFilterError
		return nil
	}
}

// WithOutputLimit clamps the controller output to the provided bounds.
func WithOutputLimit(lower, upper float64) Option {
	return func(o *options) error {
		o.outputLimit = newLimit(lower, upper)
		return nil
	}
}

// WithTrapezoidalIntegral configures whether the [*Controller] should use the
// trapezoidal method for the integral term.
//
// This method is more accurate and better suited for systems with slower
// sampling rates or that require higher precision. If set to false, the
// controller uses the simpler rectangular (Euler) method, which is generally
// sufficient for high-rate or less complex systems.
func WithTrapezoidalIntegral(enabled bool) Option {
	return func(o *options) error {
		o.trapezoidalIntegral = enabled
		return nil
	}
}

// WithOptions permits aggregating multiple options together, and is useful to
// avoid having to append options when creating helper functions or wrappers.
func WithOptions(opts ...Option) Option {
	return func(o *options) error {
		for _, opt := range opts {
			if err := opt(o); err != nil {
				return err
			}
		}
		return nil
	}
}
