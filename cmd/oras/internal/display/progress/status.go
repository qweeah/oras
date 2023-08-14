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
	"time"

	"github.com/dustin/go-humanize"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

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

	left := fmt.Sprintf("%c %s %s %s", GetMark(s), s.prompt, d, name)
	right := fmt.Sprintf(" %s/%s %.2f%% %s", humanize.Bytes(uint64(current)), humanize.Bytes(total), percent*100, s.DurationString())
	if len(left)+len(right) > width {
		right = fmt.Sprintf(" %.2f%%", percent*100)
	}
	return fmt.Sprintf("%-*s%s", width-len(right)-1, left, right)
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
