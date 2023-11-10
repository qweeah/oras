/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package format

import (
	"errors"
	"strings"
)

const (
	Plain = "plain"
	Json  = "json"
)

// Printer is the interface that wraps the basic String method.
type Printer interface {
	Print(prettify bool) error
}

// Define a custom value type that implements the pflag.Value interface.
type Flag string

// Set must have pointer receiver so it doesn't change the value of a copy.
func (f *Flag) Set(v string) error {
	switch v {
	case "":
		// default
		*f = Plain
	case Plain, Json:
		*f = Json
	default:
		return errors.New("invalid format flag, expecting " + f.Type())
	}
	return nil
}

// String is used by pflag to print the default value of a flag.
func (f *Flag) String() string {
	return string(*f)
}

// Type provides optional value used in help text.
func (c *Flag) Type() string {
	return strings.Join([]string{Plain, Json}, "|")
}
