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

package progress

import (
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// status is a progress status
type status struct {
	prompt     string
	descriptor ocispec.Descriptor
	offset     uint64
}

func NewStatus(prompt string, descriptor ocispec.Descriptor, offset uint64) *status {
	return &status{
		prompt:     prompt,
		descriptor: descriptor,
		offset:     offset,
	}
}

// String returns a viewable TTY string of the status.
func (s *status) String(width int) (string, string) {
	if s == nil {
		return "loading status...", "loading progress..."
	}
	// todo: doesn't support multiline prompt
	current := s.offset
	total := uint64(s.descriptor.Size)
	d := s.descriptor.Digest.Encoded()[:12]
	percent := float64(s.offset) / float64(total)

	name := s.descriptor.Annotations["org.opencontainers.image.title"]
	if name == "" {
		name = s.descriptor.MediaType
	}

	progress := fmt.Sprintf("] %s/%s %.2f%%", humanize.Bytes(current), humanize.Bytes(total), percent*100)

	barLen := width - len(progress)
	bar := fmt.Sprintf("   └─[%.*s", barLen-6, strings.Repeat("=", int(float64(barLen)*percent))+">")
	return fmt.Sprintf("%c %s %s %s", GetMark(s), s.prompt, d, name), fmt.Sprintf("%-*s%s", barLen, bar, progress)
}
