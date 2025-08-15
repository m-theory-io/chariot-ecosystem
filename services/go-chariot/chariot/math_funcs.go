package chariot

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
)

// RegisterMath registers all math-related functions as closures
func RegisterMath(rt *Runtime) {
	// Basic arithmetic
	rt.Register("add", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("add requires 2 arguments")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}
		if tvar, ok := args[1].(ScopeEntry); ok {
			args[1] = tvar.Value
		}

		n1, ok1 := args[0].(Number)
		n2, ok2 := args[1].(Number)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("add requires two numbers")
		}
		return Number(n1 + n2), nil
	})

	rt.Register("sub", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("sub requires 2 arguments")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}
		if tvar, ok := args[1].(ScopeEntry); ok {
			args[1] = tvar.Value
		}

		n1, ok1 := args[0].(Number)
		n2, ok2 := args[1].(Number)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("sub requires two numbers")
		}
		return Number(n1 - n2), nil
	})

	rt.Register("mul", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("mul requires 2 arguments")
		}
		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}
		if tvar, ok := args[1].(ScopeEntry); ok {
			args[1] = tvar.Value
		}
		n1, ok1 := args[0].(Number)
		n2, ok2 := args[1].(Number)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("mul requires two numbers")
		}
		return Number(n1 * n2), nil
	})

	rt.Register("div", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("div requires 2 arguments")
		}
		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}
		if tvar, ok := args[1].(ScopeEntry); ok {
			args[1] = tvar.Value
		}
		n1, ok1 := args[0].(Number)
		n2, ok2 := args[1].(Number)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("div requires two numbers")
		}
		if n2 == 0 {
			return nil, errors.New("division by zero")
		}
		return Number(n1 / n2), nil
	})

	rt.Register("mod", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("mod requires 2 arguments")
		}
		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}
		if tvar, ok := args[1].(ScopeEntry); ok {
			args[1] = tvar.Value
		}

		n1, ok1 := args[0].(Number)
		n2, ok2 := args[1].(Number)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("mod requires two numbers")
		}
		if n2 == 0 {
			return nil, errors.New("modulo by zero")
		}
		return Number(math.Mod(float64(n1), float64(n2))), nil
	})

	// Advanced math
	rt.Register("abs", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("abs requires 1 argument")
		}
		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		n, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("abs requires a number")
		}
		return Number(math.Abs(float64(n))), nil
	})

	rt.Register("sqrt", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("sqrt requires 1 argument")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		n, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("sqrt requires a number")
		}
		if n < 0 {
			return nil, errors.New("cannot take square root of negative number")
		}
		return Number(math.Sqrt(float64(n))), nil
	})

	rt.Register("pow", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("pow requires 2 arguments")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}
		if tvar, ok := args[1].(ScopeEntry); ok {
			args[1] = tvar.Value
		}

		base, ok1 := args[0].(Number)
		exp, ok2 := args[1].(Number)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("pow requires two numbers")
		}
		return Number(math.Pow(float64(base), float64(exp))), nil
	})

	rt.Register("exp", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("exp requires 1 argument")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		n, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("exp requires a number")
		}
		return Number(math.Exp(float64(n))), nil
	})

	rt.Register("floor", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("floor requires 1 argument")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		n, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("floor requires a number")
		}
		return Number(math.Floor(float64(n))), nil
	})

	rt.Register("ceiling", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("ceiling requires 1 argument")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		n, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("ceiling requires a number")
		}
		return Number(math.Ceil(float64(n))), nil
	})

	rt.Register("ceil", func(args ...Value) (Value, error) {
		// Just call the existing ceiling function
		return rt.funcs["ceiling"](args...)
	})

	rt.Register("round", func(args ...Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, errors.New("round requires 1 or 2 arguments")
		}

		// Unwrap the args as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		n, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("round requires a number")
		}
		if len(args) == 2 {
			places, ok := args[1].(Number)
			if !ok {
				return nil, fmt.Errorf("round: places must be a number")
			}
			shift := math.Pow(10, float64(places))
			return Number(math.Round(float64(n)*shift) / shift), nil
		}
		return Number(math.Round(float64(n))), nil
	})

	rt.Register("int", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("int requires 1 argument")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		n, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("int requires a number")
		}
		return Number(math.Trunc(float64(n))), nil
	})

	// Logarithmic functions
	rt.Register("log", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("log requires 1 argument")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		n, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("log requires a number")
		}
		if n <= 0 {
			return nil, errors.New("logarithm of non-positive number")
		}
		return Number(math.Log(float64(n))), nil
	})

	rt.Register("log10", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("log10 requires 1 argument")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		n, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("log10 requires a number")
		}
		if n <= 0 {
			return nil, errors.New("logarithm of non-positive number")
		}
		return Number(math.Log10(float64(n))), nil
	})

	rt.Register("log2", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("log2 requires 1 argument")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		n, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("log2 requires a number")
		}
		if n <= 0 {
			return nil, errors.New("logarithm of non-positive number")
		}
		return Number(math.Log2(float64(n))), nil
	})

	rt.Register("ln", func(args ...Value) (Value, error) {
		// Natural log - alias for log for clarity
		return rt.funcs["log"](args...)
	})

	// Trigonometric functions
	rt.Register("sin", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("sin requires 1 argument")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		n, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("sin requires a number")
		}
		return Number(math.Sin(float64(n))), nil
	})

	rt.Register("cos", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("cos requires 1 argument")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		n, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("cos requires a number")
		}
		return Number(math.Cos(float64(n))), nil
	})

	rt.Register("tan", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("tan requires 1 argument")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		n, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("tan requires a number")
		}
		return Number(math.Tan(float64(n))), nil
	})

	// Add mathematical constants
	rt.Register("pi", func(args ...Value) (Value, error) {
		if len(args) != 0 {
			return nil, errors.New("pi requires no arguments")
		}
		return Number(math.Pi), nil
	})

	rt.Register("e", func(args ...Value) (Value, error) {
		if len(args) != 0 {
			return nil, errors.New("e requires no arguments")
		}
		return Number(math.E), nil
	})

	// Add missing seed function for random - not recommended in modern Go,
	// but included for compatibility with legacy apps
	rt.Register("randomSeed", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("randomSeed requires 1 argument")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		seed, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("randomSeed requires a number")
		}
		rand.Seed(int64(seed))
		return Number(seed), nil
	})

	// Statistics
	rt.Register("max", func(args ...Value) (Value, error) {
		if len(args) < 1 {
			return nil, errors.New("max requires at least 1 argument")
		}

		// Unwrap the arguments as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		maxVal, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("max requires numbers")
		}
		for i := 1; i < len(args); i++ {
			n, ok := args[i].(Number)
			if !ok {
				return nil, fmt.Errorf("max requires numbers")
			}
			if n > maxVal {
				maxVal = n
			}
		}
		return Number(maxVal), nil
	})

	rt.Register("min", func(args ...Value) (Value, error) {
		if len(args) < 1 {
			return nil, errors.New("min requires at least 1 argument")
		}

		// Unwrap the arguments as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		minVal, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("min requires numbers")
		}
		for i := 1; i < len(args); i++ {
			n, ok := args[i].(Number)
			if !ok {
				return nil, fmt.Errorf("min requires numbers")
			}
			if n < minVal {
				minVal = n
			}
		}
		return Number(minVal), nil
	})

	rt.Register("random", func(args ...Value) (Value, error) {

		// Unwrap the arguments as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		if len(args) == 0 {
			return Number(rand.Float64()), nil
		} else if len(args) == 1 {
			max, ok := args[0].(Number)
			if !ok {
				return nil, fmt.Errorf("random: expected number")
			}
			return Number(rand.Float64() * float64(max)), nil
		} else if len(args) == 2 {
			min, ok1 := args[0].(Number)
			max, ok2 := args[1].(Number)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("random: expected numbers")
			}
			return Number(float64(min) + rand.Float64()*(float64(max)-float64(min))), nil
		}
		return nil, errors.New("random accepts 0, 1, or 2 arguments")
	})

	rt.Register("randomString", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("randomString requires 1 argument")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}

		length, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("randomString: expected number")
		}
		if length < 1 {
			return nil, errors.New("randomString: length must be positive")
		}
		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		result := make([]byte, int(length))
		for i := range result {
			result[i] = charset[rand.Intn(len(charset))]
		}
		return Str(string(result)), nil
	})

	rt.Register("sum", func(args ...Value) (Value, error) {

		// Unwrap the arguments as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		if len(args) == 0 {
			return Number(0), nil
		}
		sum := 0.0
		for _, arg := range args {
			n, ok := arg.(Number)
			if !ok {
				return nil, fmt.Errorf("sum requires numbers")
			}
			sum += float64(n)
		}
		return Number(sum), nil
	})

	rt.Register("avg", func(args ...Value) (Value, error) {
		if len(args) == 0 {
			return nil, errors.New("avg requires at least 1 argument")
		}

		// Unwrap the arguments as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		sum, err := rt.funcs["sum"](args...)
		if err != nil {
			return nil, err
		}
		return Number(float64(sum.(Number)) / float64(len(args))), nil
	})

	// Financial
	rt.Register("pct", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("pct requires 2 arguments")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}
		if tvar, ok := args[1].(ScopeEntry); ok {
			args[1] = tvar.Value
		}

		value, ok1 := args[0].(Number)
		percentage, ok2 := args[1].(Number)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("pct requires numbers")
		}
		return Number(float64(value) * float64(percentage) / 100.0), nil
	})

	rt.Register("pmt", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("pmt requires 3 arguments: principal, rate, years")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}
		if tvar, ok := args[1].(ScopeEntry); ok {
			args[1] = tvar.Value
		}
		if tvar, ok := args[2].(ScopeEntry); ok {
			args[2] = tvar.Value
		}

		principal, ok1 := args[0].(Number)
		annualRate, ok2 := args[1].(Number)
		years, ok3 := args[2].(Number)
		if !ok1 || !ok2 || !ok3 {
			return nil, fmt.Errorf("pmt requires numbers")
		}

		// Convert annual rate to monthly rate and years to months
		r := float64(annualRate) / 12.0 // monthly rate
		n := float64(years) * 12.0      // total months
		pv := float64(principal)

		if r == 0 {
			return Number(pv / n), nil
		}

		// PMT formula: PV * [r(1+r)^n] / [(1+r)^n - 1]
		numerator := pv * r * math.Pow(1+r, n)
		denominator := math.Pow(1+r, n) - 1
		payment := numerator / denominator
		// Round to 2 decimal places for financial precision
		return Number(math.Round(payment*100) / 100), nil
	})

	rt.Register("nper", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("nper requires 3 arguments: rate, pmt, pv")
		}

		// Unwrap the arguments as needed
		if tvar, ok := args[0].(ScopeEntry); ok {
			args[0] = tvar.Value
		}
		if tvar, ok := args[1].(ScopeEntry); ok {
			args[1] = tvar.Value
		}
		if tvar, ok := args[2].(ScopeEntry); ok {
			args[2] = tvar.Value
		}

		rate, ok1 := args[0].(Number)
		pmt, ok2 := args[1].(Number)
		pv, ok3 := args[2].(Number)
		if !ok1 || !ok2 || !ok3 {
			return nil, fmt.Errorf("nper requires numbers")
		}
		if rate == 0 {
			if pmt == 0 {
				return nil, errors.New("cannot calculate nper with zero rate and zero payment")
			}
			return Number(float64(pv) / float64(pmt)), nil
		}
		if pmt == 0 {
			return nil, errors.New("payment cannot be zero")
		}
		r := float64(rate)
		payment := float64(pmt)
		present := float64(pv)
		numerator := math.Log(payment / (payment - present*r))
		denominator := math.Log(1 + r)
		return Number(numerator / denominator), nil
	})

	rt.Register("rate", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("rate requires 3 arguments: nper, pmt, pv")
		}

		// Unwrap the arguments as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		nper, ok1 := args[0].(Number)
		pmt, ok2 := args[1].(Number)
		pv, ok3 := args[2].(Number)
		if !ok1 || !ok2 || !ok3 {
			return nil, fmt.Errorf("rate requires numbers")
		}

		N := float64(nper)
		PMT := float64(pmt)
		PV := float64(pv)

		if N == 0 {
			return nil, errors.New("number of periods cannot be zero")
		}
		if PMT == 0 {
			return nil, errors.New("payment cannot be zero")
		}

		// Newton-Raphson method
		const (
			maxIter   = 100
			tolerance = 1e-7
		)
		rate := 0.1 // initial guess

		for i := 0; i < maxIter; i++ {
			// f(r) = PV*(1+r)^N + PMT*((1+r)^N - 1)/r
			pow := math.Pow(1+rate, N)
			f := PV*pow + PMT*(pow-1)/rate

			// f'(r) = PV*N*(1+r)^(N-1) + PMT * [ ( (1+r)^N - 1 )/r^2 + N*(1+r)^(N-1)/r ]
			df := PV*N*math.Pow(1+rate, N-1) +
				PMT*((pow-1)/(rate*rate)+N*math.Pow(1+rate, N-1)/rate)

			if df == 0 {
				break // avoid division by zero
			}

			newRate := rate - f/df
			if math.Abs(newRate-rate) < tolerance {
				return Number(newRate), nil
			}
			rate = newRate
		}

		return Number(rate), errors.New("rate calculation did not converge")
	})
	rt.Register("irr", func(args ...Value) (Value, error) {
		if len(args) < 1 {
			return nil, errors.New("irr requires at least 1 argument")
		}

		// Unwrap the arguments as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		cashFlows := make([]float64, len(args))
		for i, arg := range args {
			cf, ok := arg.(Number)
			if !ok {
				return nil, fmt.Errorf("irr requires numbers")
			}
			cashFlows[i] = float64(cf)
		}

		const (
			maxIter   = 100
			tolerance = 1e-7
		)
		rate := 0.1 // initial guess

		for i := 0; i < maxIter; i++ {
			numerator := 0.0
			denominator := 0.0
			for t, cf := range cashFlows {
				pow := math.Pow(1+rate, float64(t))
				numerator += cf / pow
				denominator -= float64(t) * cf / pow
			}

			if math.Abs(numerator) < tolerance {
				return Number(rate), nil
			}

			if denominator == 0 {
				break // avoid division by zero
			}

			newRate := rate - numerator/denominator
			if math.Abs(newRate-rate) < tolerance {
				return Number(newRate), nil
			}
			rate = newRate
		}

		return Number(rate), errors.New("irr calculation did not converge")
	})
	rt.Register("npv", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, errors.New("npv requires at least 2 arguments: rate and cash flows")
		}

		// Unwrap the arguments as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		rate, ok := args[0].(Number)
		if !ok {
			return nil, fmt.Errorf("npv requires a number for rate")
		}
		cashFlows := make([]float64, len(args)-1)
		for i, arg := range args[1:] {
			cf, ok := arg.(Number)
			if !ok {
				return nil, fmt.Errorf("npv requires numbers for cash flows")
			}
			cashFlows[i] = float64(cf)
		}

		npv := 0.0
		for t, cf := range cashFlows {
			pow := math.Pow(1+float64(rate), float64(t))
			npv += cf / pow
		}
		return Number(npv), nil
	})
	rt.Register("fv", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("fv requires 3 arguments: rate, nper, pmt")
		}

		// Unwrap the arguments as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		rate, ok1 := args[0].(Number)
		nper, ok2 := args[1].(Number)
		pmt, ok3 := args[2].(Number)
		if !ok1 || !ok2 || !ok3 {
			return nil, fmt.Errorf("fv requires numbers")
		}
		if nper == 0 {
			return nil, errors.New("number of periods cannot be zero")
		}
		r := float64(rate)
		n := float64(nper)
		payment := float64(pmt)
		futureValue := payment * ((math.Pow(1+r, n) - 1) / r)
		return Number(futureValue), nil
	})
	rt.Register("pv", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("pv requires 3 arguments: fv, rate, nper")
		}

		// Unwrap the arguments as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		fv, ok1 := args[0].(Number)
		rate, ok2 := args[1].(Number)
		nper, ok3 := args[2].(Number)
		if !ok1 || !ok2 || !ok3 {
			return nil, fmt.Errorf("pv requires numbers")
		}
		if nper == 0 {
			return nil, errors.New("number of periods cannot be zero")
		}

		// PV = FV / (1 + r)^n
		r := float64(rate)
		n := float64(nper)
		futureValue := float64(fv)
		presentValue := futureValue / math.Pow(1+r, n)
		// Round to 2 decimal places for financial precision
		return Number(math.Round(presentValue*100) / 100), nil
	})

	rt.Register("amortize", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("amortize requires 3 arguments: rate, nper, pv")
		}

		// Unwrap the arguments as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		rate, ok1 := args[0].(Number)
		nper, ok2 := args[1].(Number)
		pv, ok3 := args[2].(Number)
		if !ok1 || !ok2 || !ok3 {
			return nil, fmt.Errorf("amortize requires numbers")
		}
		r := float64(rate)
		n := int(nper)
		balance := float64(pv)

		// Calculate payment using the PMT formula
		var payment float64
		if r == 0 {
			payment = balance / float64(n)
		} else {
			payment = balance * r * math.Pow(1+r, float64(n)) / (math.Pow(1+r, float64(n)) - 1)
		}

		// Build the schedule
		schedule := &ArrayValue{}
		for period := 1; period <= n; period++ {
			interest := balance * r
			principal := payment - interest
			if principal > balance {
				principal = balance // last payment adjustment
				payment = principal + interest
			}
			balance -= principal
			row := map[string]Value{
				"period":    Number(period),
				"payment":   Number(payment),
				"principal": Number(principal),
				"interest":  Number(interest),
				"balance":   Number(balance),
			}
			schedule.Append(row)
			if balance <= 0 {
				break
			}
		}
		return schedule, nil
	})

	rt.Register("balloon", func(args ...Value) (Value, error) {
		if len(args) != 4 {
			return nil, errors.New("balloon requires 4 arguments: rate, nper, pv, paid")
		}

		// Unwrap the arguments as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		rate, ok1 := args[0].(Number)
		nper, ok2 := args[1].(Number)
		pv, ok3 := args[2].(Number)
		paid, ok4 := args[3].(Number)
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return nil, fmt.Errorf("balloon requires numbers")
		}
		r := float64(rate)
		n := float64(nper)
		PV := float64(pv)
		k := float64(paid)
		if k > n {
			return nil, errors.New("paid periods cannot exceed total periods")
		}

		// Calculate payment using PMT formula
		var payment float64
		if r == 0 {
			payment = PV / n
		} else {
			payment = PV * r * math.Pow(1+r, n) / (math.Pow(1+r, n) - 1)
		}

		// Remaining balance after k payments
		var balloon float64
		if r == 0 {
			balloon = PV - payment*k
		} else {
			balloon = PV*math.Pow(1+r, k) - payment*(math.Pow(1+r, k)-1)/r
		}

		if balloon < 0 {
			balloon = 0
		}
		return Number(balloon), nil
	})

	rt.Register("interestOnly", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("interestOnly requires 2 arguments: rate, pv")
		}

		// Unwrap the arguments as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		rate, ok1 := args[0].(Number)
		pv, ok2 := args[1].(Number)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("interestOnly requires numbers")
		}
		interestPayment := float64(rate) * float64(pv)
		return Number(interestPayment), nil
	})

	rt.Register("interestOnlySchedule", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("interestOnlySchedule requires 3 arguments: rate, nper, pv")
		}

		// Unwrap the arguments as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		rate, ok1 := args[0].(Number)
		nper, ok2 := args[1].(Number)
		pv, ok3 := args[2].(Number)
		if !ok1 || !ok2 || !ok3 {
			return nil, fmt.Errorf("interestOnlySchedule requires numbers")
		}
		r := float64(rate)
		n := int(nper)
		balance := float64(pv)
		schedule := &ArrayValue{}
		for period := 1; period <= n; period++ {
			interest := balance * r
			row := map[string]Value{
				"period":   Number(period),
				"payment":  Number(interest),
				"interest": Number(interest),
				"balance":  Number(balance),
			}
			schedule.Append(row)
		}
		return schedule, nil
	})
	rt.Register("depreciation", func(args ...Value) (Value, error) {
		if len(args) != 4 {
			return nil, errors.New("depreciation requires 4 arguments: cost, salvage, life, method")
		}

		// Unwrap the arguments as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		cost, ok1 := args[0].(Number)
		salvage, ok2 := args[1].(Number)
		life, ok3 := args[2].(Number)
		method, ok4 := args[3].(Str)
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return nil, fmt.Errorf("depreciation requires numbers and a string for method")
		}
		var depreciation float64
		switch method {
		case "straight-line":
			depreciation = (float64(cost) - float64(salvage)) / float64(life)
		case "double-declining-balance":
			depreciation = 2 * (float64(cost) - float64(salvage)) / float64(life)
		default:
			return nil, fmt.Errorf("unsupported depreciation method: %s", method)
		}
		return Number(depreciation), nil
	})

	rt.Register("apr", func(args ...Value) (Value, error) {
		if len(args) < 2 || len(args) > 3 {
			return nil, errors.New("apr requires 2 or 3 arguments: rate, nper, [fees]")
		}

		// Unwrap the arguments as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		rate, ok1 := args[0].(Number)
		nper, ok2 := args[1].(Number)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("apr requires numbers for rate and nper")
		}
		fees := 0.0
		if len(args) == 3 {
			f, ok := args[2].(Number)
			if !ok {
				return nil, fmt.Errorf("fees must be a number")
			}
			fees = float64(f)
		}
		// Calculate APR using the formula:
		// APR = (rate * nper) + (fees / loan amount) * nper * 100
		// For simplicity, assume loan amount is 1 (so fees are a percentage)
		nominalAPR := float64(rate) * float64(nper)
		apr := nominalAPR + (fees * float64(nper))
		return Number(apr), nil
	})

	rt.Register("loanBalance", func(args ...Value) (Value, error) {
		if len(args) != 4 {
			return nil, errors.New("loanBalance requires 4 arguments: rate, nper, pv, paid")
		}

		// Unwrap the arguments as needed
		for i, arg := range args {
			if tvar, ok := arg.(ScopeEntry); ok {
				args[i] = tvar.Value
			}
		}

		rate, ok1 := args[0].(Number)
		nper, ok2 := args[1].(Number)
		pv, ok3 := args[2].(Number)
		paid, ok4 := args[3].(Number)
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return nil, fmt.Errorf("loanBalance requires numbers")
		}
		r := float64(rate)
		n := float64(nper)
		PV := float64(pv)
		k := float64(paid)
		if k > n {
			return nil, errors.New("paid periods cannot exceed total periods")
		}

		// Calculate payment using PMT formula
		var payment float64
		if r == 0 {
			payment = PV / n
		} else {
			payment = PV * r * math.Pow(1+r, n) / (math.Pow(1+r, n) - 1)
		}

		// Remaining balance after k payments
		var balance float64
		if r == 0 {
			balance = PV - payment*k
		} else {
			balance = PV*math.Pow(1+r, k) - payment*(math.Pow(1+r, k)-1)/r
		}

		if balance < 0 {
			balance = 0
		}
		return Number(balance), nil
	})
}
