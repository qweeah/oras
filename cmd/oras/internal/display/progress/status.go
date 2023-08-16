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
	"time"
	"unicode/utf8"

	"github.com/dustin/go-humanize"
	"github.com/morikuni/aec"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const BarMaxLength = 40

// status is a progress status
type status struct {
	prompt     string
	descriptor ocispec.Descriptor
	offset     int64
	startTime  *time.Time
	endTime    *time.Time
}

func NewStatus(prompt string, descriptor ocispec.Descriptor, offset uint64) *status {
	return &status{
		prompt:     prompt,
		descriptor: descriptor,
		offset:     int64(offset),
	}
}

// StartTiming starts timing.
func StartTiming() *status {
	now := time.Now()
	return &status{
		offset:    -1,
		startTime: &now,
	}
}

// EndTiming starts timing.
func EndTiming() *status {
	now := time.Now()
	return &status{
		offset:  -1,
		endTime: &now,
	}
}

// String returns a viewable TTY string of the status.
func (s *status) String(width int) (string, string) {
	if s == nil {
		return "loading status...", "loading progress..."
	}
	// todo: doesn't support multiline prompt
	total := uint64(s.descriptor.Size)
	percent := float64(s.offset) / float64(total)

	name := s.descriptor.Annotations["org.opencontainers.image.title"]
	if name == "" {
		name = s.descriptor.MediaType
	}

	// Todo: if horizontal space is not enough, hide some detail
	// format: bar(42) mark(1) action(<10) name(126)    size_per_size(19) percent(8) time(8)
	//           └─ digest(72)
	lenBar := int(percent * BarMaxLength)
	bar := fmt.Sprintf("[%s%s]", aec.Inverse.Apply(strings.Repeat(" ", lenBar)), strings.Repeat(".", BarMaxLength-lenBar))
	left := fmt.Sprintf("%s %c %s %s", bar, GetMark(s), s.prompt, name)
	right := fmt.Sprintf(" %s/%s %6.2f%% %s", humanize.Bytes(uint64(s.offset)), humanize.Bytes(total), percent*100, s.DurationString())
	return fmt.Sprintf("%-*s%s", width-utf8.RuneCountInString(right), left, right), fmt.Sprintf("  └─ %s", s.descriptor.Digest.String())
}

// DurationString returns a viewable TTY string of the status with duration.
func (s *status) DurationString() string {
	if s.startTime == nil {
		return "00:00:00"
	}

	var d time.Duration
	if s.endTime == nil {
		d = time.Since(*s.startTime)
	} else {
		d = s.endTime.Sub(*s.startTime)
	}

	if d > time.Millisecond {
		d = d.Round(time.Millisecond)
	} else {
		d = d.Round(10 * time.Nanosecond)
	}
	return d.String()
}

// Update updates a status.
func (s *status) Update(new *status) *status {
	if s == nil {
		s = &status{}
	}
	if new.offset > 0 {
		s.descriptor = new.descriptor
		s.offset = new.offset
	}
	if new.prompt != "" {
		s.prompt = new.prompt
	}
	if new.startTime != nil {
		s.startTime = new.startTime
	}
	if new.endTime != nil {
		s.endTime = new.endTime
	}
	return s
}
