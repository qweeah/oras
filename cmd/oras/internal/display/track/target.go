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
	"context"
	"io"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/display/progress"
)

type Trackable interface {
	Prompt(desc ocispec.Descriptor, prompt string)
	Stop()
}

type Target interface {
	oras.GraphTarget
	Trackable
}

type target struct {
	oras.GraphTarget
	m            progress.Manager
	actionPrompt string
	donePrompt   string
	statusMap    *sync.Map
}

func NewTarget(t oras.GraphTarget, actionPrompt, donePrompt string) (Target, error) {
	manager, err := progress.NewManager()
	if err != nil {
		return nil, err
	}

	return &target{
		GraphTarget:  t,
		m:            manager,
		actionPrompt: actionPrompt,
		donePrompt:   donePrompt,
		statusMap:    &sync.Map{},
	}, nil
}

func (t *target) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	r, err := managedReader(content, expected, t.m, t.actionPrompt)
	if err != nil {
		return err
	}
	defer close(r.status)
	if err := t.GraphTarget.Push(ctx, expected, r); err != nil {
		return err
	}
	r.status <- progress.NewStatus(t.donePrompt, expected, uint64(expected.Size))
	return nil
}

func (t *target) PushReference(ctx context.Context, expected ocispec.Descriptor, content io.Reader, reference string) error {
	r, err := managedReader(content, expected, t.m, t.actionPrompt)
	if err != nil {
		return err
	}
	defer close(r.status)
	if rp, ok := t.GraphTarget.(registry.ReferencePusher); ok {
		err = rp.PushReference(ctx, expected, r, reference)
	} else {
		if err := t.GraphTarget.Push(ctx, expected, r); err != nil {
			return err
		}
		err = t.GraphTarget.Tag(ctx, expected, reference)
	}

	if err != nil {
		return err
	}
	r.status <- progress.NewStatus(t.donePrompt, expected, uint64(expected.Size))
	return nil
}

func (t *target) Stop() {
	t.m.Wait()
}

func (t *target) Prompt(desc ocispec.Descriptor, prompt string) {
	status := t.m.Add()
	status <- progress.NewStatus(prompt, desc, uint64(desc.Size))
	defer close(status)
}
