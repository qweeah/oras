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

package match

import (
	"fmt"
	"strings"

	"github.com/onsi/gomega"
)

// status represents the expected value of first field in the status log.
type status = string

// state represents the expected value of second and third fields next status log.
type StateKey struct {
	Digest string
	Name   string
}

type state struct {
	uint // just padding to make address unique
}

type edge = struct {
	from *state
	to   *state
}

// stateMachine with edges named after known status.
type stateMachine struct {
	edges map[status][]edge
	start *state
	end   *state
}

func newGraph(cmd string) (sm *stateMachine) {
	sm = &stateMachine{
		start: new(state),
		end:   new(state),
		edges: make(map[string][]edge),
	}

	// prepare edges
	switch cmd {
	case "push", "attach":
		sm.addPath("Uploading", "Uploaded")
		sm.addPath("Exists")
		sm.addPath("Skipped")
	case "pull":
		sm.addPath("Downloading", "Downloaded")
		sm.addPath("Downloading", "Processing", "Downloaded")
		sm.addPath("Skipped")
		sm.addPath("Restored")
	default:
		panic("Unrecognized cmd name " + cmd)
	}
	return sm
}

func findState(from *state, edges []edge) *edge {
	for _, e := range edges {
		if e.from == from {
			return &e
		}
	}
	return nil
}

func (opts *stateMachine) addPath(statuses ...string) {
	last := opts.start
	len := len(statuses)
	for i, status := range statuses {
		e := findState(last, opts.edges[status])
		if e == nil {
			// new edge
			if i == len-1 {
				e = &edge{from: last, to: opts.end}
			} else {
				e = &edge{from: last, to: new(state)}
			}
			opts.edges[status] = append(opts.edges[status], *e)
		}
		last = e.to
	}
}

// statusMatcher type helps matching statusMatcher log of a oras command.
type statusMatcher struct {
	states       map[StateKey]*state
	matchResult  map[status][]StateKey
	successCount int
	verbose      bool

	*stateMachine
}

// NewStatusMatcher generates a instance for matchable status logs.
func NewStatusMatcher(keys []StateKey, cmd string, verbose bool, successCount int) *statusMatcher {
	s := statusMatcher{
		states:       make(map[StateKey]*state),
		matchResult:  make(map[string][]StateKey),
		stateMachine: newGraph(cmd),
		successCount: successCount,
		verbose:      verbose,
	}
	for _, k := range keys {
		s.states[k] = s.start
	}
	return &s
}

// switchState moves a node forward in the state machine graph.
func (s *statusMatcher) switchState(st status, key StateKey) {
	// load state
	now, ok := s.states[key]
	gomega.Expect(ok).To(gomega.BeTrue(), fmt.Sprintf("Should find state node for %v", key))

	// find next
	e := findState(now, s.edges[st])
	gomega.Expect(e).NotTo(gomega.BeNil(), fmt.Sprintf("Should state node not matching for %v, %v", st, key))

	// switch
	s.states[key] = e.to
	if e.to == s.end {
		// collect last state for matching
		s.matchResult[st] = append(s.matchResult[st], key)
	}
}

func (s *statusMatcher) Match(got []byte) {
	lines := strings.Split(string(got), "\n")
	for _, line := range lines {
		// get state key
		fields := strings.Fields(string(line))

		cnt := len(fields)
		if cnt == 2 && !s.verbose {
			// media type is hidden, add it
			fields = append(fields, "")
		}
		if cnt <= 2 || cnt > 3 {
			continue
		}
		key := StateKey{fields[1], fields[2]}
		if _, ok := s.states[key]; !ok {
			// ignore other logs
			continue
		}

		s.switchState(fields[0], key)
	}

	successCnt := 0
	for _, v := range s.matchResult {
		successCnt += len(v)
	}
	gomega.Expect(successCnt).To(gomega.Equal(s.successCount))
}
