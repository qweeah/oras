package pipe

import (
	"encoding/json"
	"fmt"
)

// Output is a generic type that can hold any data
type Output struct {
	Data interface{} `json:"data"`
}

// NewOutput creates a new Output with the given data
func NewOutput(data interface{}) *Output {
	return &Output{Data: data}
}

// Print prints the Output as JSON to the standard output
func (o *Output) String() {
	// You can use json.MarshalIndent to pretty-print the JSON output
	// The second argument is the prefix for each line
	// The third argument is the indentation for each level
	b, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	// You can use fmt.Printf or os.Stdout.Write to print the JSON bytes
	fmt.Printf("%s\n", b)
	// os.Stdout.Write(b)
}
