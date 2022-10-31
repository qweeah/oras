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

package scenario

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "oras.land/oras/test/e2e/internal/utils"
	"oras.land/oras/test/e2e/internal/utils/match"
)

var (
	artFiles = []string{
		"foobar/foo1",
		"foobar/foo2",
		"foobar/bar",
	}
	artifactTexts = []match.StateKey{
		{Digest: "2c26b46b68ff", Name: artFiles[0]},
		{Digest: "2c26b46b68ff", Name: artFiles[1]},
		{Digest: "fcde2b2edba5", Name: artFiles[2]},
	}
)

var _ = Describe("OCI image user:", Ordered, func() {
	Auth()

	repo := "oci-image"
	When("pushing images and check", func() {
		tag := "image"
		var tempDir string
		BeforeAll(func() {
			tempDir = GinkgoT().TempDir()
			if err := CopyTestData(artFiles, tempDir); err != nil {
				panic(err)
			}
		})

		It("should push and pull an image", func() {
			manifestName := "packed.json"
			ORAS("push", Reference(Host, repo, tag), "--artifact-type", "test-artifact", artFiles[0], artFiles[1], artFiles[2], "-v", "--export-manifest", manifestName).
				MatchStatus(artifactTexts, true, 3).
				WithWorkDir(tempDir).
				WithDescription("push artFiles with manifest exported").Exec()

			session := Binary("cat", manifestName).WithWorkDir(tempDir).Exec()
			ORAS("manifest", "fetch", Reference(Host, repo, tag)).
				MatchContent(string(session.Out.Contents())).
				WithDescription("fetch pushed manifest content").Exec()
			pullRoot := "pulled"
			ORAS("pull", Reference(Host, repo, tag), "-v", "-o", pullRoot).
				MatchStatus(artifactTexts, true, 2).
				WithWorkDir(tempDir).
				WithDescription("should pull artFiles with config").Exec()

			for _, f := range artFiles {
				Binary("diff", filepath.Join(f), filepath.Join(pullRoot, f)).
					WithWorkDir(tempDir).
					WithDescription("should download identical file " + f).Exec()
			}
		})

	})
})
