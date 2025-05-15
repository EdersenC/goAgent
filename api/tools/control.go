package tools

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Tool Registry
var toolRegistry = map[string]func(string) (string, error){
	"calculator": Calculate,
	"uppercase":  ToUpperCase,
}

// RegisterTool allows dynamic tool registration
func RegisterTool(name string, toolFunc func(string) (string, error)) {
	toolRegistry[name] = toolFunc
}

// UseTool executes a tool by name with the given input
func UseTool(toolName, input string) (string, error) {
	if tool, exists := toolRegistry[toolName]; exists {
		return tool(input)
	}
	return "", errors.New("tool not found")
}

// Calculator Tool (basic arithmetic parsing)
func Calculate(input string) (string, error) {
	parts := strings.Fields(input)
	if len(parts) != 3 {
		return "", errors.New("invalid input format, expected: <num1> <operator> <num2>")
	}

	num1, err1 := strconv.Atoi(parts[0])
	num2, err2 := strconv.Atoi(parts[2])
	operator := parts[1]

	if err1 != nil || err2 != nil {
		return "", errors.New("invalid numbers")
	}

	switch operator {
	case "+":
		return strconv.Itoa(num1 + num2), nil
	case "-":
		return strconv.Itoa(num1 - num2), nil
	case "*":
		return strconv.Itoa(num1 * num2), nil
	case "/":
		if num2 == 0 {
			return "", errors.New("division by zero")
		}
		return strconv.Itoa(num1 / num2), nil
	default:
		return "", errors.New("unsupported operator")
	}
}

// Uppercase Tool (converts input to uppercase)
func ToUpperCase(input string) (string, error) {
	return strings.ToUpper(input), nil
}

// Example Usage
func main() {
	// Register a new tool dynamically
	RegisterTool("reverse", func(input string) (string, error) {
		runes := []rune(input)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes), nil
	})

	// Test tools
	results := []struct {
		tool  string
		input string
	}{
		{"calculator", "5 + 3"},
		{"uppercase", "hello world"},
		{"reverse", "golang"},
		{"nonexistent", "test"},
	}

	for _, test := range results {
		output, err := UseTool(test.tool, test.input)
		if err != nil {
			fmt.Printf("Error using %s: %v\n", test.tool, err)
		} else {
			fmt.Printf("%s(%q) -> %q\n", test.tool, test.input, output)
		}
	}
}






