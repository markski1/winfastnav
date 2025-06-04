package utils

import (
	"errors"
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
		if strings.ContainsRune("+-/* ", r) {
			hasRune = true
			continue
		}
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return hasRune
}

func EvalMath(expr string) (float64, error) {
	tokens, err := tokenize(expr)
	if err != nil {
		return 0, err
	}

	// Infix and Postfix notation is pretty cool and worth reading about.
	// Basically: The way in which humans write operations (Infix: operand, operator, operand, operator [repeat for as many operands])
	// is quite hard to compute compared to Postfix ([]operands, operator, []operands, operator)
	// So you just put the operators in proper order and go through their operands. Neat!
	postfix, err := infixToPostfix(tokens)
	if err != nil {
		return 0, err
	}

	return evalPostfix(postfix)
}

/*
	A significant chunk of the code below is LLM generated.
*/

func tokenize(expr string) ([]string, error) {
	var tokens []string
	var number strings.Builder
	for i, r := range expr {
		if unicode.IsDigit(r) {
			number.WriteRune(r)
		} else if strings.ContainsRune("+-/*", r) {
			if number.Len() > 0 {
				tokens = append(tokens, number.String())
				number.Reset()
			}
			tokens = append(tokens, string(r))
		} else {
			return nil, errors.New("invalid character in expression")
		}

		// If last char and number buffer is not empty, flush it
		if i == len(expr)-1 && number.Len() > 0 {
			tokens = append(tokens, number.String())
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
		if _, err := strconv.Atoi(token); err == nil {
			// Token is a number
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
