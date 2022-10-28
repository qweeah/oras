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

package display

import (
	"context"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
)

// UploadOption returns the copy graph option for upload-related commands.
func UploadOption(source content.Fetcher, committed *sync.Map, concurrency int64, verbose bool, blobs []ocispec.Descriptor) oras.CopyGraphOptions {
	graphCopyOptions := oras.DefaultCopyGraphOptions
	graphCopyOptions.Concurrency = concurrency
	graphCopyOptions.PreCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		if _, loaded := committed.LoadOrStore(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle]); !loaded {
			return PrintStatus(desc, "Uploading", verbose)
		}
		return nil
	}
	graphCopyOptions.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
		if _, loaded := committed.LoadOrStore(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle]); !loaded {
			return PrintStatus(desc, "Exists   ", verbose)
		}
		return nil
	}
	graphCopyOptions.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		if err := PrintSuccessorStatus(ctx, desc, "Skipped  ", source, committed, verbose); err != nil {
			return err
		}
		return PrintStatus(desc, "Uploaded ", verbose)
	}

	graphCopyOptions.FindSuccessors = func(ctx context.Context, fetcher content.Fetcher, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		successors, err := content.Successors(ctx, source, desc)
		if err != nil {
			return nil, err
		}
		start := 0
		for i, s := range successors {
			if _, done := committed.Load(s.Digest.String()); done {
				// swap and slice committed node off
				successors[i] = successors[start]
				start++
			}
		}
		return successors[start:], nil
	}

	return graphCopyOptions
}
