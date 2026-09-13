package utils

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

func ConvertUnit(s string) string {
	if result := convertDirectedUnit(s); result != "" {
		return result
	}
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")

	i := 0
	if i < len(s) && (s[i] == '-' || s[i] == '+') {
		i++
	}
	for i < len(s) && (s[i] == '.' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	if i == 0 || i == len(s) {
		return ""
	}
	numStr, unit := s[:i], s[i:]
	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return ""
	}

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
	case "celsius":
		unit = "c"
	case "fahrenheit":
		unit = "f"
	case "kelvin":
		unit = "k"
	case "kph", "kmh":
		unit = "km/h"
	case "mps", "ms-1":
		unit = "m/s"
	case "fps", "ft/s":
		unit = "ft/s"
	}

	var out []string
	formatNumber := func(f float64) string {
		s := fmt.Sprintf("%.2f", f)
		if strings.HasSuffix(s, ".00") {
			return strings.TrimSuffix(s, ".00")
		}
		return s
	}

	switch unit {
	case "kg":
		out = append(out, formatNumber(val*1000)+" g")
		out = append(out, formatNumber(val*2.2046226218)+" lb")
		out = append(out, formatNumber(val*35.27396195)+" oz")
	case "g":
		kg := val / 1000
		out = append(out, formatNumber(kg)+" kg")
		out = append(out, formatNumber(kg*2.2046226218)+" lb")
		out = append(out, formatNumber(kg*35.27396195)+" oz")
	case "lb":
		kg := val / 2.2046226218
		out = append(out, formatNumber(kg)+" kg")
		out = append(out, formatNumber(kg*1000)+" g")
		out = append(out, formatNumber(val*16)+" oz")
	case "oz":
		lb := val / 16
		kg := lb / 2.2046226218
		out = append(out, formatNumber(kg)+" kg")
		out = append(out, formatNumber(kg*1000)+" g")
		out = append(out, formatNumber(lb)+" lb")

	case "m":
		out = append(out, formatNumber(val*100)+" cm")
		out = append(out, formatNumber(val*1000)+" mm")
		out = append(out, formatNumber(val*39.37007874)+" in")
		out = append(out, formatNumber(val*3.280839895)+" ft")
	case "cm":
		m := val / 100
		out = append(out, formatNumber(m)+" m")
		out = append(out, formatNumber(val*10)+" mm")
		out = append(out, formatNumber(m*39.37007874)+" in")
		out = append(out, formatNumber(m*3.280839895)+" ft")
	case "mm":
		cm := val / 10
		m := cm / 100
		out = append(out, formatNumber(m)+" m")
		out = append(out, formatNumber(cm)+" cm")
		out = append(out, formatNumber(m*39.37007874)+" in")
		out = append(out, formatNumber(m*3.280839895)+" ft")
	case "in":
		m := val / 39.37007874
		out = append(out, formatNumber(m)+" m")
		out = append(out, formatNumber(m*100)+" cm")
		out = append(out, formatNumber(m*1000)+" mm")
		out = append(out, formatNumber(val/12)+" ft")
	case "ft":
		m := val / 3.280839895
		out = append(out, formatNumber(m)+" m")
		out = append(out, formatNumber(m*100)+" cm")
		out = append(out, formatNumber(m*1000)+" mm")
		out = append(out, formatNumber(val*12)+" in")

	case "c":
		c := val
		f := c*9.0/5.0 + 32.0
		k := c + 273.15
		out = append(out, formatNumber(f)+" °F")
		out = append(out, formatNumber(k)+" K")
	case "f":
		fv := val
		c := (fv - 32.0) * 5.0 / 9.0
		k := c + 273.15
		out = append(out, formatNumber(c)+" °C")
		out = append(out, formatNumber(k)+" K")
	case "k":
		kv := val
		c := kv - 273.15
		f := c*9.0/5.0 + 32.0
		out = append(out, formatNumber(c)+" °C")
		out = append(out, formatNumber(f)+" °F")

	case "m/s":
		ms := val
		kmh := ms * 3.6
		mph := ms * 2.2369362921
		fts := ms * 3.280839895
		out = append(out, formatNumber(kmh)+" kmh")
		out = append(out, formatNumber(mph)+" mph")
		out = append(out, formatNumber(fts)+" fps")
	case "km/h":
		kmh := val
		ms := kmh / 3.6
		mph := kmh * 0.6213711922
		fts := ms * 3.280839895
		out = append(out, formatNumber(ms)+" m/s")
		out = append(out, formatNumber(mph)+" mph")
		out = append(out, formatNumber(fts)+" fps")
	case "mph":
		mph := val
		kmh := mph * 1.609344
		ms := kmh / 3.6
		fts := ms * 3.280839895
		out = append(out, formatNumber(ms)+" m/s")
		out = append(out, formatNumber(kmh)+" kmh")
		out = append(out, formatNumber(fts)+" fps")
	case "ft/s":
		fts := val
		ms := fts / 3.280839895
		kmh := ms * 3.6
		mph := ms * 2.2369362921
		out = append(out, formatNumber(ms)+" m/s")
		out = append(out, formatNumber(kmh)+" kmh")
		out = append(out, formatNumber(mph)+" mph")

	default:
		return ""
	}

	return strings.Join(out, "\n")
}

type unitDefinition struct {
	group   string
	factor  float64
	display string
}

var directedUnits = map[string]unitDefinition{
	"kg": {"mass", 1, "kg"}, "kilogram": {"mass", 1, "kg"}, "kilograms": {"mass", 1, "kg"},
	"g": {"mass", 0.001, "g"}, "gram": {"mass", 0.001, "g"}, "grams": {"mass", 0.001, "g"},
	"lb": {"mass", 0.45359237, "lb"}, "lbs": {"mass", 0.45359237, "lb"}, "pound": {"mass", 0.45359237, "lb"}, "pounds": {"mass", 0.45359237, "lb"},
	"oz": {"mass", 0.028349523125, "oz"}, "ounce": {"mass", 0.028349523125, "oz"}, "ounces": {"mass", 0.028349523125, "oz"},
	"m": {"length", 1, "m"}, "meter": {"length", 1, "m"}, "meters": {"length", 1, "m"}, "metre": {"length", 1, "m"}, "metres": {"length", 1, "m"},
	"km": {"length", 1000, "km"}, "kilometer": {"length", 1000, "km"}, "kilometers": {"length", 1000, "km"},
	"cm": {"length", 0.01, "cm"}, "centimeter": {"length", 0.01, "cm"}, "centimeters": {"length", 0.01, "cm"},
	"mm": {"length", 0.001, "mm"}, "millimeter": {"length", 0.001, "mm"}, "millimeters": {"length", 0.001, "mm"},
	"in": {"length", 0.0254, "in"}, "inch": {"length", 0.0254, "in"}, "inches": {"length", 0.0254, "in"},
	"ft": {"length", 0.3048, "ft"}, "foot": {"length", 0.3048, "ft"}, "feet": {"length", 0.3048, "ft"},
	"yd": {"length", 0.9144, "yd"}, "yard": {"length", 0.9144, "yd"}, "yards": {"length", 0.9144, "yd"},
	"mi": {"length", 1609.344, "mi"}, "mile": {"length", 1609.344, "mi"}, "miles": {"length", 1609.344, "mi"},
	"m/s": {"speed", 1, "m/s"}, "mps": {"speed", 1, "m/s"},
	"km/h": {"speed", 1 / 3.6, "km/h"}, "kmh": {"speed", 1 / 3.6, "km/h"}, "kph": {"speed", 1 / 3.6, "km/h"},
	"mph": {"speed", 0.44704, "mph"}, "ft/s": {"speed", 0.3048, "ft/s"}, "fps": {"speed", 0.3048, "ft/s"},
	"knot": {"speed", 0.514444444444, "kn"}, "knots": {"speed", 0.514444444444, "kn"}, "kn": {"speed", 0.514444444444, "kn"},
	"l": {"volume", 1, "L"}, "liter": {"volume", 1, "L"}, "liters": {"volume", 1, "L"}, "litre": {"volume", 1, "L"}, "litres": {"volume", 1, "L"},
	"ml": {"volume", 0.001, "mL"}, "gallon": {"volume", 3.785411784, "gal"}, "gallons": {"volume", 3.785411784, "gal"}, "gal": {"volume", 3.785411784, "gal"},
	"qt": {"volume", 0.946352946, "qt"}, "quart": {"volume", 0.946352946, "qt"}, "quarts": {"volume", 0.946352946, "qt"},
	"b": {"data", 1, "B"}, "byte": {"data", 1, "B"}, "bytes": {"data", 1, "B"},
	"kb": {"data", 1000, "KB"}, "mb": {"data", 1e6, "MB"}, "gb": {"data", 1e9, "GB"}, "tb": {"data", 1e12, "TB"},
	"kib": {"data", 1024, "KiB"}, "mib": {"data", 1048576, "MiB"}, "gib": {"data", 1073741824, "GiB"}, "tib": {"data", 1099511627776, "TiB"},
	"ms": {"duration", 0.001, "ms"}, "second": {"duration", 1, "s"}, "seconds": {"duration", 1, "s"}, "sec": {"duration", 1, "s"}, "s": {"duration", 1, "s"},
	"minute": {"duration", 60, "min"}, "minutes": {"duration", 60, "min"}, "min": {"duration", 60, "min"},
	"hour": {"duration", 3600, "h"}, "hours": {"duration", 3600, "h"}, "hr": {"duration", 3600, "h"}, "h": {"duration", 3600, "h"},
	"day": {"duration", 86400, "days"}, "days": {"duration", 86400, "days"},
	"c": {"temperature", 1, "°C"}, "celsius": {"temperature", 1, "°C"},
	"f": {"temperature", 1, "°F"}, "fahrenheit": {"temperature", 1, "°F"},
	"k": {"temperature", 1, "K"}, "kelvin": {"temperature", 1, "K"},
}

func convertDirectedUnit(input string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(input)), " "))
	separator := " to "
	index := strings.LastIndex(normalized, separator)
	if index < 0 {
		separator = " in "
		index = strings.LastIndex(normalized, separator)
	}
	if index < 0 {
		return ""
	}
	value, sourceName, ok := parseQuantity(normalized[:index])
	if !ok {
		return ""
	}
	targetName := strings.TrimSpace(normalized[index+len(separator):])
	source, sourceOK := directedUnits[sourceName]
	target, targetOK := directedUnits[targetName]
	if !sourceOK || !targetOK || source.group != target.group {
		return ""
	}
	if source.group == "temperature" {
		converted, ok := convertTemperature(value, sourceName, targetName)
		if !ok {
			return ""
		}
		return formatNumber(converted) + " " + target.display
	}
	return formatNumber(value*source.factor/target.factor) + " " + target.display
}

func parseQuantity(input string) (float64, string, bool) {
	input = strings.TrimSpace(input)
	index := 0
	if index < len(input) && (input[index] == '-' || input[index] == '+') {
		index++
	}
	for index < len(input) && ((input[index] >= '0' && input[index] <= '9') || input[index] == '.' || input[index] == ',') {
		index++
	}
	if index == 0 || index == len(input) {
		return 0, "", false
	}
	value, ok := parseNumber(input[:index])
	unit := strings.TrimSpace(input[index:])
	return value, unit, ok && unit != ""
}

func convertTemperature(value float64, source, target string) (float64, bool) {
	source = directedUnits[source].display
	target = directedUnits[target].display
	celsius := value
	switch source {
	case "°F":
		celsius = (value - 32) * 5 / 9
	case "K":
		celsius = value - 273.15
	case "°C":
	default:
		return 0, false
	}
	switch target {
	case "°C":
		return celsius, true
	case "°F":
		return celsius*9/5 + 32, true
	case "K":
		return celsius + 273.15, true
	default:
		return 0, false
	}
}

func IsMath(s string) bool {
	if len(s) == 0 {
		return false
	}
	hasRune := false
	for _, r := range s {
		if strings.ContainsRune("+-/*^()., ", r) {
			if strings.ContainsRune("+-/*^()", r) {
				hasRune = true
			}
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
	parser := expressionParser{input: expr}
	result, err := parser.expression()
	parser.skipSpaces()
	if err != nil || parser.position != len(parser.input) || math.IsInf(result, 0) || math.IsNaN(result) {
		if err == nil {
			err = errors.New("invalid expression")
		}
		return "0", err
	}
	return formatNumber(result), nil
}

type expressionParser struct {
	input    string
	position int
}

func (p *expressionParser) expression() (float64, error) {
	left, err := p.term()
	for err == nil {
		p.skipSpaces()
		if p.position >= len(p.input) || (p.input[p.position] != '+' && p.input[p.position] != '-') {
			break
		}
		operator := p.input[p.position]
		p.position++
		var right float64
		right, err = p.term()
		if operator == '+' {
			left += right
		} else {
			left -= right
		}
	}
	return left, err
}

func (p *expressionParser) term() (float64, error) {
	left, err := p.unary()
	for err == nil {
		p.skipSpaces()
		if p.position >= len(p.input) || (p.input[p.position] != '*' && p.input[p.position] != '/') {
			break
		}
		operator := p.input[p.position]
		p.position++
		var right float64
		right, err = p.unary()
		if operator == '*' {
			left *= right
		} else if right == 0 {
			return 0, errors.New("division by zero")
		} else {
			left /= right
		}
	}
	return left, err
}

func (p *expressionParser) power() (float64, error) {
	left, err := p.primary()
	if err != nil {
		return 0, err
	}
	p.skipSpaces()
	if p.position < len(p.input) && p.input[p.position] == '^' {
		p.position++
		right, err := p.unary()
		if err != nil {
			return 0, err
		}
		left = math.Pow(left, right)
	}
	return left, nil
}

func (p *expressionParser) unary() (float64, error) {
	p.skipSpaces()
	if p.position < len(p.input) && (p.input[p.position] == '+' || p.input[p.position] == '-') {
		operator := p.input[p.position]
		p.position++
		value, err := p.unary()
		if operator == '-' {
			value = -value
		}
		return value, err
	}
	return p.power()
}

func (p *expressionParser) primary() (float64, error) {
	p.skipSpaces()
	if p.position >= len(p.input) {
		return 0, errors.New("missing operand")
	}
	if p.input[p.position] == '(' {
		p.position++
		value, err := p.expression()
		p.skipSpaces()
		if err != nil || p.position >= len(p.input) || p.input[p.position] != ')' {
			return 0, errors.New("unclosed parenthesis")
		}
		p.position++
		return value, nil
	}
	start := p.position
	dot := false
	for p.position < len(p.input) {
		character := p.input[p.position]
		if character >= '0' && character <= '9' {
			p.position++
			continue
		}
		if character == '.' && !dot {
			dot = true
			p.position++
			continue
		}
		break
	}
	if start == p.position {
		return 0, errors.New("missing number")
	}
	return strconv.ParseFloat(p.input[start:p.position], 64)
}

func (p *expressionParser) skipSpaces() {
	for p.position < len(p.input) && unicode.IsSpace(rune(p.input[p.position])) {
		p.position++
	}
}
