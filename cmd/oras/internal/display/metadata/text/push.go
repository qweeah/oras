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

package text

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/view"
	"oras.land/oras/cmd/oras/internal/option"
)

// PushHandler handles text metadata output for push events.
type PushHandler struct {
	view.Printer
}

// NewPushHandler returns a new handler for push events.
func NewPushHandler() metadata.PushHandler {
	return &PushHandler{
		Printer: view.NewPrinter(),
	}
}

// OnCopied is called after files are copied.
func (p *PushHandler) OnCopied(opts *option.Target) error {
	_, err := p.Println("Pushed", opts.AnnotatedReference())
	return err
}

// OnCompleted is called after the push is completed.
func (p *PushHandler) OnCompleted(root ocispec.Descriptor) error {
	_, err := p.Println("Digest:", root.Digest)
	return err
}
