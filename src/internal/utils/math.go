package utils

/*
	To my dismay, this source file is largely LLM generated.
	I had a go at doing most of this myself but I just kept digging into
	deeper rabbit holes, tokenization is a hell of a drug.
*/

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

func HasUnit(s string) bool {
	if len(s) == 0 {
		return false
	}
	s = strings.ToLower(s)
	units := []string{
		"kg", "g",
		"lb", "lbs", "pound",
		"oz", "ounce", "ounces",
		"cm", "mm", "m",
		"in", "inch", "inches",
		"ft", "foot", "feet",
		// temperature
		"c", "celsius",
		"f", "fahrenheit",
		"k", "kelvin",
		// speed
		"m/s", "mps", "ms-1", "m·s-1",
		"km/h", "kph", "kmh",
		"mph",
		"ft/s", "fps",
	}

	for _, u := range units {
		if strings.Contains(s, u) {
			return true
		}
	}
	return false
}

func ConvertUnit(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, ",", "")
	// allow for spaces by just ignoring them
	s = strings.ReplaceAll(s, " ", "")

	// grab the number
	i := 0
	for i < len(s) && (s[i] == '.' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	// return if no number
	if i == 0 || i == len(s) {
		return ""
	}
	numStr, unit := s[:i], s[i:]
	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil || val < 0 {
		return ""
	}

	// normalize unit synonyms
	switch unit {
	case "pound", "pounds":
		unit = "lb"
	case "ounce", "ounces":
		unit = "oz"
	case "inch", "inches":
		unit = "in"
	case "foot", "feet":
		unit = "ft"
	case "lbs":
		unit = "lb"
	// temperature synonyms
	case "celsius":
		unit = "c"
	case "fahrenheit":
		unit = "f"
	case "kelvin":
		unit = "k"
	// speed synonyms
	case "kph", "kmh":
		unit = "km/h"
	case "mps", "ms-1", "m·s-1":
		unit = "m/s"
	case "fps", "ft/s":
		unit = "ft/s"
	}

	var out []string
	f2 := func(f float64) string {
		// trim .00
		s := fmt.Sprintf("%.2f", f)
		if strings.HasSuffix(s, ".00") {
			return strings.TrimSuffix(s, ".00")
		}
		return s
	}

	switch unit {
	// mass conversions
	case "kg":
		out = append(out, f2(val*1000)+" g")
		out = append(out, f2(val*2.2046226218)+" lb")
		out = append(out, f2(val*35.27396195)+" oz")
	case "g":
		kg := val / 1000
		out = append(out, f2(kg)+" kg")
		out = append(out, f2(kg*2.2046226218)+" lb")
		out = append(out, f2(kg*35.27396195)+" oz")
	case "lb":
		kg := val / 2.2046226218
		out = append(out, f2(kg)+" kg")
		out = append(out, f2(kg*1000)+" g")
		out = append(out, f2(val*16)+" oz")
	case "oz":
		lb := val / 16
		kg := lb / 2.2046226218
		out = append(out, f2(kg)+" kg")
		out = append(out, f2(kg*1000)+" g")
		out = append(out, f2(lb)+" lb")

	// length conversions
	case "m":
		out = append(out, f2(val*100)+" cm")
		out = append(out, f2(val*1000)+" mm")
		out = append(out, f2(val*39.37007874)+" in")
		out = append(out, f2(val*3.280839895)+" ft")
	case "cm":
		m := val / 100
		out = append(out, f2(m)+" m")
		out = append(out, f2(val*10)+" mm")
		out = append(out, f2(m*39.37007874)+" in")
		out = append(out, f2(m*3.280839895)+" ft")
	case "mm":
		cm := val / 10
		m := cm / 100
		out = append(out, f2(m)+" m")
		out = append(out, f2(cm)+" cm")
		out = append(out, f2(m*39.37007874)+" in")
		out = append(out, f2(m*3.280839895)+" ft")
	case "in":
		m := val / 39.37007874
		out = append(out, f2(m)+" m")
		out = append(out, f2(m*100)+" cm")
		out = append(out, f2(m*1000)+" mm")
		out = append(out, f2(val/12)+" ft")
	case "ft":
		m := val / 3.280839895
		out = append(out, f2(m)+" m")
		out = append(out, f2(m*100)+" cm")
		out = append(out, f2(m*1000)+" mm")
		out = append(out, f2(val*12)+" in")

	// temperature conversions
	case "c":
		c := val
		f := c*9.0/5.0 + 32.0
		k := c + 273.15
		out = append(out, f2(f)+" °F")
		out = append(out, f2(k)+" K")
	case "f":
		fv := val
		c := (fv - 32.0) * 5.0 / 9.0
		k := c + 273.15
		out = append(out, f2(c)+" °C")
		out = append(out, f2(k)+" K")
	case "k":
		kv := val
		c := kv - 273.15
		f := c*9.0/5.0 + 32.0
		out = append(out, f2(c)+" °C")
		out = append(out, f2(f)+" °F")

	// speed conversions
	case "m/s":
		ms := val
		kmh := ms * 3.6
		mph := ms * 2.2369362921
		fts := ms * 3.280839895
		out = append(out, f2(kmh)+" kmh")
		out = append(out, f2(mph)+" mph")
		out = append(out, f2(fts)+" fps")
	case "km/h":
		kmh := val
		ms := kmh / 3.6
		mph := kmh * 0.6213711922
		fts := ms * 3.280839895
		out = append(out, f2(ms)+" m/s")
		out = append(out, f2(mph)+" mph")
		out = append(out, f2(fts)+" fps")
	case "mph":
		mph := val
		kmh := mph * 1.609344
		ms := kmh / 3.6
		fts := ms * 3.280839895
		out = append(out, f2(ms)+" m/s")
		out = append(out, f2(kmh)+" kmh")
		out = append(out, f2(fts)+" fps")
	case "ft/s":
		fts := val
		ms := fts / 3.280839895
		kmh := ms * 3.6
		mph := ms * 2.2369362921
		out = append(out, f2(ms)+" m/s")
		out = append(out, f2(kmh)+" kmh")
		out = append(out, f2(mph)+" mph")

	default:
		return ""
	}

	return strings.Join(out, "\n")
}

func IsMath(s string) bool {
	if len(s) == 0 {
		return false
	}
	hasRune := false
	for _, r := range s {
		if strings.ContainsRune("+-/*,. ", r) {
			hasRune = true
			continue
		}
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return hasRune
}

func EvalMath(expr string) (string, error) {
	expr = strings.ReplaceAll(expr, ",", ".")
	tokens, err := tokenize(expr)
	if err != nil {
		return "0", err
	}

	// Infix and Postfix notation is pretty cool and worth reading about.
	// Basically: The way in which humans write operations (Infix: operand, operator, operand, operator [repeat for as many operands])
	// is quite hard to compute compared to Postfix ([]operands, operator, []operands, operator)
	// So you just put the operators in proper order and go through their operands. Neat!
	postfix, err := infixToPostfix(tokens)
	if err != nil {
		return "0", err
	}

	result, err := evalPostfix(postfix)

	if err != nil {
		return "0", err
	}

	strResult := fmt.Sprintf("%.2f", result)
	return strings.ReplaceAll(strResult, ".00", ""), nil
}

func tokenize(expr string) ([]string, error) {
	var tokens []string
	var number strings.Builder
	dotCount := 0

	flushNumber := func() error {
		if number.Len() > 0 {
			// Check that number is valid float
			numStr := number.String()
			if strings.Count(numStr, ".") > 1 {
				return errors.New("invalid number with multiple decimal points: " + numStr)
			}
			tokens = append(tokens, numStr)
			number.Reset()
			dotCount = 0
		}
		return nil
	}

	for i, r := range expr {
		if unicode.IsDigit(r) {
			number.WriteRune(r)
		} else if r == '.' {
			if dotCount >= 1 {
				return nil, errors.New("invalid number with multiple decimal points")
			}
			dotCount++
			number.WriteRune(r)
		} else if strings.ContainsRune("+-/*", r) {
			if err := flushNumber(); err != nil {
				return nil, err
			}
			tokens = append(tokens, string(r))
		} else if unicode.IsSpace(r) {
			// On space flush number (if any)
			if err := flushNumber(); err != nil {
				return nil, err
			}
			// Skip spaces otherwise
		} else {
			return nil, errors.New("invalid character in expression")
		}

		// If last char and number buffer is not empty, flush it
		if i == len(expr)-1 {
			if err := flushNumber(); err != nil {
				return nil, err
			}
		}
	}
	return tokens, nil
}

var precedence = map[string]int{
	"+": 1,
	"-": 1,
	"*": 2,
	"/": 2,
}

func infixToPostfix(tokens []string) ([]string, error) {
	var output []string
	var stack []string

	for _, token := range tokens {
		if _, err := strconv.ParseFloat(token, 64); err == nil {
			// Token is a number (float supported)
			output = append(output, token)
		} else if p, ok := precedence[token]; ok {
			// Token is operator
			for len(stack) > 0 {
				top := stack[len(stack)-1]
				if topPrecedence, ok := precedence[top]; ok && topPrecedence >= p {
					// Pop from stack to output while stack top has >= precedence
					output = append(output, top)
					stack = stack[:len(stack)-1]
				} else {
					break
				}
			}
			stack = append(stack, token)
		} else {
			return nil, errors.New("invalid token: " + token)
		}
	}

	// Pop remaining operators
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		output = append(output, top)
	}

	return output, nil
}

func evalPostfix(tokens []string) (float64, error) {
	var stack []float64
	for _, token := range tokens {
		if val, err := strconv.ParseFloat(token, 64); err == nil {
			stack = append(stack, val)
		} else {
			// Operator: pop last two values
			if len(stack) < 2 {
				return 0, errors.New("invalid expression: insufficient operands")
			}
			b := stack[len(stack)-1]
			a := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			var res float64
			switch token {
			case "+":
				res = a + b
			case "-":
				res = a - b
			case "*":
				res = a * b
			case "/":
				if b == 0 {
					return 0, errors.New("division by zero")
				}
				res = a / b
			default:
				return 0, errors.New("unknown operator: " + token)
			}
			stack = append(stack, res)
		}
	}
	if len(stack) != 1 {
		return 0, errors.New("invalid expression")
	}
	return stack[0], nil
}
