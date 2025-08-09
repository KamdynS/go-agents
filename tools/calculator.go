package tools

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
)

// CalculatorTool provides simple arithmetic: add, sub, mul, div, pow, sqrt.
// Input format: "op arg1 [arg2]" e.g., "add 1 2", "sqrt 9"
type CalculatorTool struct{}

func (c *CalculatorTool) Name() string { return "calculator" }
func (c *CalculatorTool) Description() string {
	return "Perform basic arithmetic. Usage: 'op arg1 [arg2]'. ops: add, sub, mul, div, pow, sqrt"
}

func (c *CalculatorTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{"input": map[string]interface{}{"type": "string"}},
		"required":   []string{"input"},
	}
}

func (c *CalculatorTool) Execute(ctx context.Context, input string) (string, error) {
	parts := strings.Fields(input)
	if len(parts) < 2 {
		return "", errors.New("usage: '<op> arg1 [arg2]'")
	}
	op := strings.ToLower(parts[0])
	parse := func(s string) (float64, error) { return strconv.ParseFloat(s, 64) }

	switch op {
	case "sqrt":
		if len(parts) != 2 {
			return "", errors.New("sqrt requires 1 argument")
		}
		a, err := parse(parts[1])
		if err != nil {
			return "", err
		}
		if a < 0 {
			return "", errors.New("sqrt of negative")
		}
		return strconv.FormatFloat(math.Sqrt(a), 'f', -1, 64), nil
	case "add", "sub", "mul", "div", "pow":
		if len(parts) != 3 {
			return "", errors.New(op + " requires 2 arguments")
		}
		a, err := parse(parts[1])
		if err != nil {
			return "", err
		}
		b, err := parse(parts[2])
		if err != nil {
			return "", err
		}
		var res float64
		switch op {
		case "add":
			res = a + b
		case "sub":
			res = a - b
		case "mul":
			res = a * b
		case "div":
			if b == 0 {
				return "", errors.New("division by zero")
			}
			res = a / b
		case "pow":
			res = math.Pow(a, b)
		}
		return strconv.FormatFloat(res, 'f', -1, 64), nil
	default:
		return "", errors.New("unknown op")
	}
}

var _ Tool = (*CalculatorTool)(nil)
