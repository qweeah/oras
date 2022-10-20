package display

import (
	"context"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
)

func SetupUploadPrinter(source content.Fetcher, concurrency int64, verbose bool, blobs []ocispec.Descriptor) oras.CopyGraphOptions {
	committed := &sync.Map{}
	graphCopyOptions := oras.DefaultCopyGraphOptions
	graphCopyOptions.Concurrency = concurrency
	graphCopyOptions.FindSuccessors = func(ctx context.Context, fetcher content.Fetcher, node ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		// if isEqualOCIDescriptor(node, root) {
		// 	// optimize find successor & skip subject
		// 	return blobs, nil
		// }
		return content.Successors(ctx, fetcher, node)
	}
	graphCopyOptions.PreCopy = StatusPrinter("Uploading", verbose)
	graphCopyOptions.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		return PrintStatus(desc, "Exists   ", verbose)
	}
	graphCopyOptions.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		if err := PrintSuccessorStatus(ctx, desc, "Skipped  ", source, committed, verbose); err != nil {
			return err
		}
		return PrintStatus(desc, "Uploaded ", verbose)
	}
	return graphCopyOptions
}

func isEqualOCIDescriptor(a, b ocispec.Descriptor) bool {
	return a.Size == b.Size && a.Digest == b.Digest && a.MediaType == b.MediaType
}
