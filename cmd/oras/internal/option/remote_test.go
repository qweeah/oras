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

package option

import (
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	nhttp "net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/pflag"
	"oras.land/oras-go/v2/registry/remote/auth"
)

var ts *httptest.Server
var testRepo = "test-repo"
var testTagList = struct {
	Tags []string `json:"tags"`
}{
	Tags: []string{"tag"},
}

func TestMain(m *testing.M) {
	// Test server
	ts = httptest.NewTLSServer(nhttp.HandlerFunc(func(w nhttp.ResponseWriter, r *nhttp.Request) {
		p := r.URL.Path
		m := r.Method
		switch {
		case p == "/v2/" && m == "GET":
			w.WriteHeader(nhttp.StatusOK)
		case p == fmt.Sprintf("/v2/%s/tags/list", testRepo) && m == "GET":
			json.NewEncoder(w).Encode(testTagList)
		}
	}))
	defer ts.Close()
	m.Run()
}

func TestRemote_FlagsInit(t *testing.T) {
	var test struct {
		Remote
	}

	ApplyFlags(&test, pflag.NewFlagSet("oras-test", pflag.ExitOnError))
}

func TestRemote_authClient_RawCredential(t *testing.T) {
	password := make([]byte, 12)
	if _, err := rand.Read(password); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := auth.Credential{
		Username: "mocked^^??oras-@@!#",
		Password: base64.StdEncoding.EncodeToString(password),
	}
	opts := Remote{
		Username: want.Username,
		Password: want.Password,
	}
	client, err := opts.authClient("hostname", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := client.Credential(nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Username != want.Username || got.Password != want.Password {
		t.Fatalf("expect: %v, got: %v", want, got)
	}
}

func TestRemote_authClient_skipTlsVerify(t *testing.T) {
	opts := Remote{
		Insecure: true,
	}
	client, err := opts.authClient("hostname", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req, err := nhttp.NewRequestWithContext(context.Background(), nhttp.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemote_authClient_CARoots(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ts.Certificate().Raw}), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(ts.Certificate())

	opts := Remote{
		CACertFilePath: caPath,
	}
	client, err := opts.authClient("hostname", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req, err := nhttp.NewRequestWithContext(context.Background(), nhttp.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemote_NewRegistry(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ts.Certificate().Raw}), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(ts.Certificate())

	opts := struct {
		Remote
		Common
	}{
		Remote{
			CACertFilePath: caPath,
		},
		Common{},
	}
	uri, err := url.ParseRequestURI(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	reg, err := opts.NewRegistry(uri.Host, opts.Common)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err = reg.Ping(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemote_NewRepository(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ts.Certificate().Raw}), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(ts.Certificate())
	opts := struct {
		Remote
		Common
	}{
		Remote{
			CACertFilePath: caPath,
		},
		Common{},
	}

	uri, err := url.ParseRequestURI(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	repo, err := opts.NewRepository(uri.Host+"/"+testRepo, opts.Common)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err = repo.Tags(context.Background(), "", func(got []string) error {
		want := []string{"tag"}
		if len(got) != len(testTagList.Tags) || !reflect.DeepEqual(got, want) {
			return fmt.Errorf("expect: %v, got: %v", testTagList.Tags, got)
		}
		return nil
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemote_isPlainHttp_localhost(t *testing.T) {
	opts := Remote{PlainHTTP: false}
	got := opts.isPlainHttp("localhost")
	if got != true {
		t.Fatalf("tls should be disabled when domain is localhost")

	}

	got = opts.isPlainHttp("localhost:9090")
	if got != true {
		t.Fatalf("tls should be disabled when domain is localhost")

	}
}

func TestRemote_ParseResolve_err(t *testing.T) {
	tests := []struct {
		name    string
		opts    *Remote
		wantErr bool
	}{
		{
			name:    "invalid host",
			opts:    &Remote{resolveFlag: []string{":port:address"}},
			wantErr: true,
		},
		{
			name:    "invalid address",
			opts:    &Remote{resolveFlag: []string{"host:port:"}},
			wantErr: true,
		},
		{
			name:    "invalid port",
			opts:    &Remote{resolveFlag: []string{"host::address"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.opts.ParseResolve(); (err != nil) != tt.wantErr {
				t.Errorf("Remote.ParseResolve() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
func TestRemote_ParseResolve_defaultFlag(t *testing.T) {
	opts := &Remote{resolveFlag: nil}
	if err := opts.ParseResolve(); err != nil {
		t.Fatalf("should succeed parsing empty resolve flag but got %v", err)
	}
	if len(opts.Resolves) != 0 {
		t.Fatalf("expect empty resolve entries but got %v", opts.Resolves)
	}
}

func TestRemote_ParseResolve_ipv4(t *testing.T) {
	host := "mockedHost"
	port := 12345
	address := "192.168.1.1"
	opts := &Remote{resolveFlag: []string{fmt.Sprintf("%s:%d:%s", host, port, address)}}
	if err := opts.ParseResolve(); err != nil {
		t.Fatalf("should succeed parsing resolve flag but got %v", err)
	}
	if len(opts.Resolves) != 1 {
		t.Fatalf("expect 1 resolve entries but got %v", opts.Resolves)
	}

	entry := opts.Resolves[0]
	if entry.from != host {
		t.Fatalf("expect resolved host %q but got %q", host, entry.from)
	}
	if entry.to.To4().String() != address {
		t.Fatalf("expect resolved address %q but got %q", address, entry.to)
	}
	if entry.port != port {
		t.Fatalf("expect resolved port %d but port %d", port, entry.port)
	}
}
