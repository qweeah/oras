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
	pushFiles = []string{
		"foobar/foo1",
		"foobar/foo2",
		"foobar/bar",
	}
	artifactPushTexts = []match.StateKey{
		{Digest: "2c26b46b68ff", Name: pushFiles[0]},
		{Digest: "2c26b46b68ff", Name: pushFiles[1]},
		{Digest: "fcde2b2edba5", Name: pushFiles[2]},
	}

	attachFile  = "foobar/attached"
	attachTexts = []match.StateKey{{Digest: "2c26b46b68ff", Name: attachFile}}
)

var _ = Describe("OCI artifact user:", Ordered, func() {
	Auth()

	repo := "oci-artifact"
	When("pushing images and check", func() {
		tag := "artifact"
		var tempDir string
		BeforeAll(func() {
			tempDir = GinkgoT().TempDir()
			if err := CopyTestData(pushFiles, tempDir); err != nil {
				panic(err)
			}
		})

		It("should push and pull an artifact", func() {
			manifestName := "packed.json"
			ORAS("push", Reference(Host, repo, tag), "--artifact-type", "test-artifact", pushFiles[0], pushFiles[1], pushFiles[2], "-v", "--export-manifest", manifestName).
				MatchStatus(artifactPushTexts, true, 3).
				WithWorkDir(tempDir).
				WithDescription("push with manifest exported").Exec()

			session := Binary("cat", manifestName).WithWorkDir(tempDir).Exec()
			ORAS("manifest", "fetch", Reference(Host, repo, tag)).
				MatchContent(string(session.Out.Contents())).
				WithDescription("fetch pushed manifest content").Exec()
			pullRoot := "pulled"
			ORAS("pull", Reference(Host, repo, tag), "-v", "-o", pullRoot).
				MatchStatus(artifactPushTexts, true, 3).
				WithWorkDir(tempDir).
				WithDescription("pull artFiles with config").Exec()

			for _, f := range pushFiles {
				Binary("diff", filepath.Join(f), filepath.Join(pullRoot, f)).
					WithWorkDir(tempDir).
					WithDescription("download identical file " + f).Exec()
			}

			ORAS("attach", Reference(Host, repo, tag), "--artifact-type", "test-artifact", "-v", attachFile, "-v", "--export-manifest", manifestName).
				MatchStatus(attachTexts, true, 1).
				WithWorkDir(tempDir).
				WithDescription("attach with manifest exported").Exec()
			session = ORAS("discover", Reference(Host, repo, tag), "-o", "json").Exec()
			dgst := Binary("jq", "-r", ".manifests[].digest").
				WithInput(session.Out).Exec().Out.Contents()

			session = Binary("cat", manifestName).WithWorkDir(tempDir).Exec()
			ORAS("manifest", "fetch", Reference(Host, repo, string(dgst))).
				MatchContent(string(session.Out.Contents())).
				WithDescription("fetch pushed manifest content").Exec()
			ORAS("pull", Reference(Host, repo, tag), "-v", "-o", pullRoot).
				MatchStatus([]match.StateKey{
					{Digest: "2c26b46b68ff", Name: pushFiles[0]},
					{Digest: "2c26b46b68ff", Name: pushFiles[1]},
					{Digest: "fcde2b2edba5", Name: pushFiles[2]},
					{Digest: "2c26b46b68ff", Name: attachFile},
				}, true, 4).
				WithWorkDir(tempDir).
				WithDescription("pull artFiles with config").Exec()
			Binary("diff", filepath.Join(attachFile), filepath.Join(pullRoot, attachFile)).
				WithWorkDir(tempDir).
				WithDescription("download identical file " + attachFile).Exec()
		})

	})
})
