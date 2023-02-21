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

package utils

import (
	"fmt"

	"github.com/opencontainers/go-digest"
)

// LayoutRef generates the reference string from given parameters.
func LayoutRef(rootPath string, tagOrDigest string) string {
	var delimiter string
	if _, err := digest.Parse(tagOrDigest); err == nil {
		// digest
		delimiter = "@"
	} else {
		// tag
		delimiter = ":"
	}
	return fmt.Sprintf("%s%s%s", rootPath, delimiter, tagOrDigest)
}
