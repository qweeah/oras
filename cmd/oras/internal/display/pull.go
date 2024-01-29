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
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/display/track"
	"oras.land/oras/cmd/oras/internal/metadata"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/graph"
)

type pullResult struct {
	files     []metadata.File
	filesLock sync.Mutex
}

// PullHandler provides display handler for pulling.
type PullHandler struct {
	template           string
	needTextOutput     bool
	verbose            bool
	includeSubject     bool
	configPath         string
	configMediaType    string
	fetcher            content.Fetcher
	tty                *os.File
	trackedGraphTarget track.GraphTarget
	printed            *sync.Map
	result             pullResult
	outputFolder       string
	target             *option.Target
	getConfigOnce      sync.Once
	layerSkipped       atomic.Bool

	promptDownloading string
	promptPulled      string
	promptProcessing  string
	promptSkipped     string
	promptRestored    string
	promptDownloaded  string
}

// NewPullHandler creates a new pull handler.
func NewPullHandler(template string, tty *os.File, fetcher content.Fetcher, verbose bool, includeSubject bool, configPath, configMediaType string, outputFolder string, target *option.Target) *PullHandler {
	ph := &PullHandler{
		template:        template,
		needTextOutput:  NeedTextOutput(template, tty),
		verbose:         verbose,
		includeSubject:  includeSubject,
		configPath:      configPath,
		configMediaType: configMediaType,
		tty:             tty,
		fetcher:         fetcher,
		printed:         &sync.Map{},
		outputFolder:    outputFolder,
		target:          target,

		promptDownloading: "Downloading",
		promptPulled:      "Pulled     ",
		promptProcessing:  "Processing ",
		promptSkipped:     "Skipped    ",
		promptRestored:    "Restored   ",
		promptDownloaded:  "Downloaded ",
	}
	if tracked, ok := fetcher.(track.GraphTarget); ok {
		ph.trackedGraphTarget = tracked
	}
	return ph
}

// FindSuccessors finds successors of a descriptor.
func (ph *PullHandler) FindSuccessors(ctx context.Context, fetcher content.Fetcher, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
	statusFetcher := content.FetcherFunc(func(ctx context.Context, target ocispec.Descriptor) (fetched io.ReadCloser, fetchErr error) {
		if _, ok := ph.printed.LoadOrStore(generateContentKey(target), true); ok {
			return fetcher.Fetch(ctx, target)
		}
		if ph.trackedGraphTarget == nil && ph.needTextOutput {
			// none TTY, print status log for first-time fetching
			if err := PrintStatus(target, ph.promptDownloading, ph.verbose); err != nil {
				return nil, err
			}
		}
		rc, err := fetcher.Fetch(ctx, target)
		if err != nil {
			return nil, err
		}
		defer func() {
			if fetchErr != nil {
				rc.Close()
			}
		}()
		if ph.trackedGraphTarget == nil && ph.needTextOutput {
			// none TTY, add logs for processing manifest
			return rc, PrintStatus(target, ph.promptProcessing, ph.verbose)
		}
		return rc, nil
	})

	nodes, subject, config, err := graph.Successors(ctx, statusFetcher, desc)
	if err != nil {
		return nil, err
	}
	if subject != nil && ph.includeSubject {
		nodes = append(nodes, *subject)
	}
	if config != nil {
		ph.getConfigOnce.Do(func() {
			if ph.configPath != "" && (ph.configMediaType == "" || config.MediaType == ph.configMediaType) {
				if config.Annotations == nil {
					config.Annotations = make(map[string]string)
				}
				config.Annotations[ocispec.AnnotationTitle] = ph.configPath
			}
		})
		if config.Size != ocispec.DescriptorEmptyJSON.Size || config.Digest != ocispec.DescriptorEmptyJSON.Digest || config.Annotations[ocispec.AnnotationTitle] != "" {
			nodes = append(nodes, *config)
		}
	}

	var ret []ocispec.Descriptor
	for _, s := range nodes {
		if s.Annotations[ocispec.AnnotationTitle] == "" {
			if content.Equal(s, ocispec.DescriptorEmptyJSON) {
				// empty layer
				continue
			}
			if s.Annotations[ocispec.AnnotationTitle] == "" {
				// unnamed layers are skipped
				ph.layerSkipped.Store(true)
			}
			ss, err := content.Successors(ctx, fetcher, s)
			if err != nil {
				return nil, err
			}
			if len(ss) == 0 {
				if err := ph.printOnce(s, ph.promptSkipped); err != nil {
					return nil, err
				}
				continue
			}
		}
		ret = append(ret, s)
	}

	return ret, nil
}

func (ph *PullHandler) printOnce(s ocispec.Descriptor, msg string) error {
	if _, loaded := ph.printed.LoadOrStore(generateContentKey(s), true); loaded {
		return nil
	}
	if ph.trackedGraphTarget != nil {
		// TTY
		return ph.trackedGraphTarget.Prompt(s, msg)
	} else if ph.needTextOutput {
		// none TTY
		return PrintStatus(s, msg, ph.verbose)
	}
	return nil
}

// generateContentKey generates a unique key for each content descriptor, using
// its digest and name if applicable.
func generateContentKey(desc ocispec.Descriptor) string {
	return desc.Digest.String() + desc.Annotations[ocispec.AnnotationTitle]
}

func (ph *PullHandler) PreCopy(ctx context.Context, desc ocispec.Descriptor) error {
	if _, ok := ph.printed.LoadOrStore(generateContentKey(desc), true); ok {
		return nil
	}
	if ph.needTextOutput {
		// none TTY, print status log for downloading
		return PrintStatus(desc, ph.promptDownloading, ph.verbose)
	}
	return nil
}

// PostCopy is called after a content is copied.
func (ph *PullHandler) PostCopy(ctx context.Context, desc ocispec.Descriptor) error {
	// restore named but deduplicated successor nodes
	successors, err := content.Successors(ctx, ph.fetcher, desc)
	if err != nil {
		return err
	}
	for _, s := range successors {
		if name, ok := s.Annotations[ocispec.AnnotationTitle]; ok {
			ph.result.filesLock.Lock()
			ph.result.files = append(ph.result.files, metadata.NewFile(name, ph.outputFolder, s, ph.target.Path))
			ph.result.filesLock.Unlock()
			if err := ph.printOnce(s, ph.promptRestored); err != nil {
				return err
			}
		}
	}
	name, ok := desc.Annotations[ocispec.AnnotationTitle]
	if !ok {
		if !ph.verbose {
			return nil
		}
		name = desc.MediaType
	}
	ph.printed.Store(generateContentKey(desc), true)
	if ph.needTextOutput {
		// none TTY, print status log for downloaded
		return Print(ph.promptDownloaded, ShortDigest(desc), name)
	}
	return nil
}

// PostPull is called after pulling.
func (ph *PullHandler) PostPull(root ocispec.Descriptor) error {
	if ph.template != "" {
		return option.WriteMetadata(ph.template, os.Stdout, metadata.NewPull(fmt.Sprintf("%s@%s", ph.target.Path, root.Digest), ph.result.files))
	} else if ph.needTextOutput {
		// suggest oras copy for pulling layers without annotation
		if ph.layerSkipped.Load() {
			Print("Skipped pulling layers without file name in", ocispec.AnnotationTitle)
			Print("Use 'oras copy", ph.target.RawReference, "--to-oci-layout <layout-dir>' to pull all layers.")
		} else {
			Print("Pulled", ph.target.AnnotatedReference())
			Print("Digest:", root.Digest)
		}
	}
	return nil
}
