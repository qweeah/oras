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

package track

import (
	"io"
	"sync"
	"sync/atomic"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/progress"
)

type reader struct {
	base         io.Reader
	offset       atomic.Uint64
	actionPrompt string
	descriptor   ocispec.Descriptor
	mu           sync.Mutex
	m            progress.Manager
	status       progress.Status
}

func NewReader(r io.Reader, descriptor ocispec.Descriptor, prompt string) (*reader, error) {
	manager, err := progress.NewManager()
	if err != nil {
		return nil, err
	}
	return managedReader(r, descriptor, manager, prompt)
}

func managedReader(r io.Reader, descriptor ocispec.Descriptor, manager progress.Manager, prompt string) (*reader, error) {
	return &reader{
		base:         r,
		descriptor:   descriptor,
		actionPrompt: prompt,
		m:            manager,
		status:       manager.Add(),
	}, nil
}

// Stop stops the status channel and related manager.
func (r *reader) Stop() {
	close(r.status)
	r.m.Wait()
}

func (r *reader) Read(p []byte) (int, error) {
	n, err := r.base.Read(p)
	if err != nil && err != io.EOF {
		return n, err
	}

	offset := r.offset.Add(uint64(n))
	if err == io.EOF {
		if offset != uint64(r.descriptor.Size) {
			return n, io.ErrUnexpectedEOF
		}
		r.mu.Lock()
		defer r.mu.Unlock()
		r.status <- progress.NewStatus(r.actionPrompt, r.descriptor, offset)
	}

	if r.mu.TryLock() {
		defer r.mu.Unlock()
		if len(r.status) < progress.BUFFER_SIZE {
			// intermediate progress might be ignored if buffer is full
			r.status <- progress.NewStatus(r.actionPrompt, r.descriptor, offset)
		}
	}
	return n, err
}

func (r *reader) Prompt(desc ocispec.Descriptor, prompt string) {
	r.status <- progress.NewStatus(prompt, desc, uint64(desc.Size))
}