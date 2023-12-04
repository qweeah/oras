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
	"path/filepath"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type File struct {
	Path string `json:"path"`
	Descriptor
}

// NewFile creates a new file metadata.
func NewFile(name string, outputDir string, desc ocispec.Descriptor, descPath string) File {
	path := name
	if !filepath.IsAbs(name) {
		// ignore error since it's successfully written to file store
		path, _ = filepath.Abs(filepath.Join(outputDir, name))
	}
	return File{
		Path:       path,
		Descriptor: ToDescriptor(descPath, desc),
	}
}

// Metadata for push command
type pull struct {
	DigestReference
	Files []File `json:"files"`
}

func NewPull(digestReference string, files []File) pull {
	return pull{
		DigestReference: DigestReference{
			Reference: digestReference,
		},
		Files: files,
	}
}
