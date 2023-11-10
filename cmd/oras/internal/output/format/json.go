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
	"encoding/json"
	"os"
)

// JSON is a Output that prints the data as JSON.
type JSON struct {
	Data interface{}
}

// NewJSON creates a new Output with the given data
func NewJSON(data interface{}) *JSON {
	return &JSON{Data: data}
}

// Print prints the Output as JSON.
func Print(data interface{}, prettify bool) error {
	var content []byte
	var err error
	if prettify {
		content, err = json.MarshalIndent(data, "", "  ")
	} else {
		content, err = json.Marshal(data)
	}
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(content)
	return err
}
