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
