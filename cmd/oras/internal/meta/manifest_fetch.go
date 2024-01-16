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

package meta

import (
	"reflect"
	"strings"
)

// ToMappable converts any struct into a map[string]interface{} with camel-case
// field names.
func ToMappable(v reflect.Value) any {
	valueKind := v.Kind()
	switch valueKind {
	case reflect.Struct:
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		elem := ToMappable(v.Elem())
		return &elem
	default:
		return v.Interface()
	}
	t := v.Type()
	numField := t.NumField()
	ret := make(map[string]any)
	for i := 0; i < numField; i++ {
		v := v.Field(i)
		tag := t.Field(i).Tag.Get("json")
		if tag == "" {
			continue
		}
		key, _, _ := strings.Cut(tag, ",")
		ret[key] = ToMappable(v)
	}
	return ret
}
