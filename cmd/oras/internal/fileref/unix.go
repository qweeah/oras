//go:build !windows

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

package fileref

import "strings"

// Parse parses file reference on unix.
func Parse(reference string, mediaType string) (filePath, mediatype string, err error) {
	i := strings.LastIndex(reference, ":")
	if i < 0 {
		return reference, mediaType, nil
	}
	return reference[:i], reference[i+1:], nil
}