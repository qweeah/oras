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
	"github.com/morikuni/aec"
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
func (s *status) String(width int) string {
	if s == nil {
		return "loading..."
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

	// format: [bar(13)] [info] [detail(algin right)]
	lenBar := int(percent / 0.05)
	bar := fmt.Sprintf("[%s%s]", aec.Inverse.Apply(strings.Repeat(" ", lenBar)), strings.Repeat(".", 20-lenBar))
	info := fmt.Sprintf("%c %s %s %s", GetMark(s), s.prompt, d, name)
	detail := fmt.Sprintf(" %s/%s %.2f%%", humanize.Bytes(current), humanize.Bytes(total), percent*100)

	left := fmt.Sprintf("%s %s", bar, info)
	if len(left)+len(detail) > width {
		// todo: show partial info when space is not enough
		detail = fmt.Sprintf(" %.2f%%", percent*100)
	}
	return fmt.Sprintf("%-*s%s", width-len(detail)-1, left, detail)
}
