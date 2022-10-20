package display

import (
	"context"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
)

func UploadCopyOption(source content.Fetcher, committed *sync.Map, concurrency int64, verbose bool, blobs []ocispec.Descriptor) oras.CopyGraphOptions {
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
	return graphCopyOptions
}
