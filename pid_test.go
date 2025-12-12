package pid

import (
	"math"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestController(t *testing.T) {
	tests := []struct {
		name           string
		opts           []Option
		target         float64
		inputs         []float64
		wantOutputs    []float64
		wantController *Controller
	}{
		{
			name: "proportional-term-is-proportional-to-the-error",
			opts: []Option{
				WithProportionalGain(1.5),
				WithIntegralGain(0.0),
				WithDerivativeGain(0.0),
			},
			target:      10,
			inputs:      []float64{7},
			wantOutputs: []float64{1.5 * (10 - 7)},
			wantController: &Controller{
				proportionalGain: 1.5,
				prevControlError: 3,
				derivative:       3,
				integral:         3,
				outputLimit:      limit{lower: math.Inf(-1), upper: math.Inf(1)},
				integralLimit:    limit{lower: math.Inf(-1), upper: math.Inf(1)},
			},
		},
		{
			name: "integral-term-is-proportional-to-magnitude-and-duration-of-error",
			opts: []Option{
				WithProportionalGain(0.0),
				WithIntegralGain(1.5),
				WithDerivativeGain(0.0),
			},
			target: 10.0,
			inputs: []float64{7, 8},
			wantOutputs: []float64{
				4.5,
				7.5,
			},
			wantController: &Controller{
				integralGain:     1.5,
				prevControlError: 2,
				integral:         5,
				derivative:       -1,
				outputLimit:      limit{lower: math.Inf(-1), upper: math.Inf(1)},
				integralLimit:    limit{lower: math.Inf(-1), upper: math.Inf(1)},
			},
		},
		{
			name: "derivative-term-responds-to-error-change",
			opts: []Option{
				WithProportionalGain(0.0),
				WithIntegralGain(0.0),
				WithDerivativeGain(1.5),
			},
			target: 10.0,
			inputs: []float64{7, 2},
			wantOutputs: []float64{
				4.5,
				7.5,
			},
			wantController: &Controller{
				derivativeGain:   1.5,
				prevControlError: 8,
				integral:         11,
				derivative:       5,
				outputLimit:      limit{lower: math.Inf(-1), upper: math.Inf(1)},
				integralLimit:    limit{lower: math.Inf(-1), upper: math.Inf(1)},
			},
		},
		{
			name: "full-pid-computes-from-proportional-integral-and-derivative-terms",
			opts: []Option{
				WithProportionalGain(2.0),
				WithIntegralGain(1.0),
				WithDerivativeGain(0.5),
			},
			target: 10.0,
			inputs: []float64{7, 8},
			wantOutputs: []float64{
				10.5,
				8.5,
			},
			wantController: &Controller{
				proportionalGain: 2.0,
				integralGain:     1.0,
				derivativeGain:   0.5,
				prevControlError: 2,
				integral:         5,
				derivative:       -1,
				outputLimit:      limit{lower: math.Inf(-1), upper: math.Inf(1)},
				integralLimit:    limit{lower: math.Inf(-1), upper: math.Inf(1)},
			},
		},
		{
			name: "zero-gains-yield-zero-output",
			opts: []Option{
				WithProportionalGain(0),
				WithIntegralGain(0),
				WithDerivativeGain(0),
			},
			target:      10.0,
			inputs:      []float64{0},
			wantOutputs: []float64{0},
			wantController: &Controller{
				prevControlError: 10,
				integral:         10,
				derivative:       10,
				outputLimit:      limit{lower: math.Inf(-1), upper: math.Inf(1)},
				integralLimit:    limit{lower: math.Inf(-1), upper: math.Inf(1)},
			},
		},
		{
			name: "output-limit-is-enforced",
			opts: []Option{
				WithProportionalGain(1.5),
				WithIntegralGain(0.0),
				WithDerivativeGain(0.0),
				WithOutputLimit(-3, 3),
			},
			target:      10,
			inputs:      []float64{7},
			wantOutputs: []float64{3},
			wantController: &Controller{
				proportionalGain: 1.5,
				prevControlError: 3,
				derivative:       3,
				integral:         3,
				outputLimit:      limit{lower: -3, upper: 3},
				integralLimit:    limit{lower: math.Inf(-1), upper: math.Inf(1)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pid, err := New(tt.opts...)
			if err != nil {
				t.Fatal(err)
			}
			if len(tt.inputs) != len(tt.wantOutputs) {
				t.Fatalf("invalid test case: inputs %d != outputs %d", len(tt.inputs), len(tt.wantOutputs))
			}

			var got []float64
			for _, current := range tt.inputs {
				output := pid.Update(tt.target, current, 1*time.Second)
				got = append(got, output)
			}
			if diff := cmp.Diff(got, tt.wantOutputs); diff != "" {
				t.Errorf("diff: %s", diff)
			}
			if diff := cmp.Diff(pid, tt.wantController, cmp.AllowUnexported(Controller{}, limit{})); diff != "" {
				t.Errorf("diff: %s", diff)
			}
		})
	}
}
