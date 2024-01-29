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

package root

import (
	"context"
	"errors"
	"fmt"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/display/track"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/fileref"
	"oras.land/oras/cmd/oras/internal/option"
)

type pullOptions struct {
	option.Cache
	option.Common
	option.Platform
	option.Target
	option.Format

	concurrency       int
	KeepOldFiles      bool
	IncludeSubject    bool
	PathTraversal     bool
	Output            string
	ManifestConfigRef string
}

func pullCmd() *cobra.Command {
	var opts pullOptions
	cmd := &cobra.Command{
		Use:   "pull [flags] <name>{:<tag>|@<digest>}",
		Short: "Pull files from a registry or an OCI image layout",
		Long: `Pull files from a registry or an OCI image layout

Example - Pull artifact files from a registry:
  oras pull localhost:5000/hello:v1

Example - Recursively pulling all files from a registry, including subjects of hello:v1:
  oras pull --include-subject localhost:5000/hello:v1

Example - Pull files from an insecure registry:
  oras pull --insecure localhost:5000/hello:v1

Example - Pull files from the HTTP registry:
  oras pull --plain-http localhost:5000/hello:v1

Example - Pull files from a registry with local cache:
  export ORAS_CACHE=~/.oras/cache
  oras pull localhost:5000/hello:v1

Example - Pull files from a registry with certain platform:
  oras pull --platform linux/arm/v5 localhost:5000/hello:v1

Example - Pull all files with concurrency level tuned:
  oras pull --concurrency 6 localhost:5000/hello:v1

Example - Pull artifact files from an OCI image layout folder 'layout-dir':
  oras pull --oci-layout layout-dir:v1

Example - Pull artifact files from an OCI layout archive 'layout.tar':
  oras pull --oci-layout layout.tar:v1
`,
		Args: oerrors.CheckArgs(argument.Exactly(1), "the artifact reference you want to pull"),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.RawReference = args[0]
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPull(cmd, &opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.KeepOldFiles, "keep-old-files", "k", false, "do not replace existing files when pulling, treat them as errors")
	cmd.Flags().BoolVarP(&opts.PathTraversal, "allow-path-traversal", "T", false, "allow storing files out of the output directory")
	cmd.Flags().BoolVarP(&opts.IncludeSubject, "include-subject", "", false, "[Preview] recursively pull the subject of artifacts")
	cmd.Flags().StringVarP(&opts.Output, "output", "o", ".", "output directory")
	cmd.Flags().StringVarP(&opts.ManifestConfigRef, "config", "", "", "output manifest config file")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 3, "concurrency level")
	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.Target)
}

func runPull(cmd *cobra.Command, opts *pullOptions) error {
	ctx, logger := opts.WithContext(cmd.Context())
	// Copy Options
	copyOptions := oras.DefaultCopyOptions
	copyOptions.Concurrency = opts.concurrency
	if opts.Platform.Platform != nil {
		copyOptions.WithTargetPlatform(opts.Platform.Platform)
	}
	target, err := opts.NewReadonlyTarget(ctx, opts.Common, logger)
	if err != nil {
		return err
	}
	if err := opts.EnsureReferenceNotEmpty(cmd, true); err != nil {
		return err
	}
	src, err := opts.CachedTarget(target)
	if err != nil {
		return err
	}
	dst, err := file.New(opts.Output)
	if err != nil {
		return err
	}
	defer dst.Close()
	dst.AllowPathTraversalOnWrite = opts.PathTraversal
	dst.DisableOverwrite = opts.KeepOldFiles

	var configPath, configMediaType string
	if opts.ManifestConfigRef != "" {
		configPath, configMediaType, err = fileref.Parse(opts.ManifestConfigRef, "")
		if err != nil {
			return err
		}
	}

	const (
		promptDownloading = "Downloading"
		promptPulled      = "Pulled     "
	)

	tracked, err := getTrackedTarget(dst, opts.TTY, promptDownloading, promptPulled)
	if err != nil {
		return err
	}
	ph := display.NewPullHandler(opts.Template, opts.TTY, tracked, opts.Verbose, opts.IncludeSubject, configPath, configMediaType, opts.Output, &opts.Target)
	copyOptions.FindSuccessors = ph.FindSuccessors
	copyOptions.PreCopy = ph.PostCopy
	copyOptions.PostCopy = ph.PostCopy

	root, err := doPull(ctx, src, tracked, copyOptions, opts)
	if err != nil {
		if errors.Is(err, file.ErrPathTraversalDisallowed) {
			err = fmt.Errorf("%s: %w", "use flag --allow-path-traversal to allow insecurely pulling files outside of working directory", err)
		}
		return err
	}
	return ph.PostPull(root)

}

func doPull(ctx context.Context, src oras.ReadOnlyTarget, dst oras.GraphTarget, opts oras.CopyOptions, po *pullOptions) (ocispec.Descriptor, error) {
	if tracked, ok := dst.(track.GraphTarget); ok {
		defer tracked.Close()
	}
	// Copy
	return oras.Copy(ctx, src, po.Reference, dst, po.Reference, opts)
}

// generateContentKey generates a unique key for each content descriptor, using
// its digest and name if applicable.
func generateContentKey(desc ocispec.Descriptor) string {
	return desc.Digest.String() + desc.Annotations[ocispec.AnnotationTitle]
}

func printOnce(printed *sync.Map, s ocispec.Descriptor, msg string, verbose bool, dst any) error {
	if _, loaded := printed.LoadOrStore(generateContentKey(s), true); loaded {
		return nil
	}
	if tracked, ok := dst.(track.GraphTarget); ok {
		// TTY
		return tracked.Prompt(s, msg)

	}
	// none TTY
	return display.PrintStatus(s, msg, verbose)
}
