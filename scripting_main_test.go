package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Exercise real process exit statuses and stdout/stderr separation without
// building another binary or depending on an external service.
func TestCLIHelperProcess(t *testing.T) {
	if os.Getenv("GO_CURLING_HELPER_PROCESS") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{os.Args[0]}, os.Args[i+1:]...)
			break
		}
	}
	main()
	os.Exit(0)
}

func TestScriptProcessContracts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/error" {
			w.WriteHeader(404)
		}
		fmt.Fprint(w, "body")
	}))
	defer srv.Close()
	for _, tc := range []struct {
		name                  string
		args                  []string
		code                  int
		output, errorContains string
	}{
		{"silent response", []string{"-s", srv.URL + "/error"}, 0, "body", ""},
		{"fail diagnostics", []string{"-f", srv.URL + "/error"}, 22, "", "404"},
		{"silent fail", []string{"-sf", srv.URL + "/error"}, 22, "", ""},
		{"show errors", []string{"-sfS", srv.URL + "/error"}, 22, "", "404"},
		{"fail body metrics", []string{"-sS", "--fail-with-body", "-w", ":%{exitcode}", srv.URL + "/error"}, 22, "body:22", "404"},
		{"verbose separation", []string{"-v", srv.URL}, 0, "body", "HTTP/1.1"},
		{"missing config", []string{"--config"}, 2, "", "requires a filename"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], append([]string{"-test.run=^TestCLIHelperProcess$", "--"}, tc.args...)...)
			cmd.Env = append(os.Environ(), "GO_CURLING_HELPER_PROCESS=1")
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			if tc.code == 0 {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.Equal(t, tc.code, cmd.ProcessState.ExitCode())
			require.Equal(t, tc.output, stdout.String())
			if tc.errorContains == "" {
				require.Empty(t, stderr.String())
			} else {
				require.Contains(t, stderr.String(), tc.errorContains)
			}
		})
	}
}

func TestStdinPipelineProcess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.Copy(w, r.Body) }))
	defer srv.Close()
	for _, args := range [][]string{{"--data-binary", "@-"}, {"--json", "@-"}, {"-T", "-"}} {
		cmd := exec.Command(os.Args[0], append(append([]string{"-test.run=^TestCLIHelperProcess$", "--"}, args...), srv.URL)...)
		cmd.Env = append(os.Environ(), "GO_CURLING_HELPER_PROCESS=1")
		cmd.Stdin = strings.NewReader("{\"input\":true}\n")
		body, err := cmd.Output()
		require.NoError(t, err)
		require.Equal(t, "{\"input\":true}\n", string(body))
	}
}
