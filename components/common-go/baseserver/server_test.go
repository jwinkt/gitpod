// Copyright (c) 2022 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package baseserver_test

import (
	"fmt"
	"github.com/gitpod-io/gitpod/common-go/baseserver"
	"github.com/gitpod-io/gitpod/common-go/certtest"
	"github.com/gitpod-io/gitpod/common-go/pprof"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

func TestServer_StartStop(t *testing.T) {
	// We don't use the helper NewForTests, because we want to control stopping ourselves.
	srv, err := baseserver.New("server_test", baseserver.WithHTTPPort(8765), baseserver.WithGRPCPort(8766))
	require.NoError(t, err)
	baseserver.StartServerForTests(t, srv)

	require.Equal(t, "http://localhost:8765", srv.HTTPAddress())
	require.Equal(t, "localhost:8766", srv.GRPCAddress())
	require.NoError(t, srv.Close())
}

func TestServer_ServesHealthEndpoints(t *testing.T) {
	for _, scenario := range []struct {
		name     string
		endpoint string
	}{
		{name: "ready endpoint", endpoint: "/ready"},
		{name: "live endpoint", endpoint: "/live"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			srv := baseserver.NewForTests(t)
			baseserver.StartServerForTests(t, srv)

			resp, err := http.Get(srv.HTTPAddress() + scenario.endpoint)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

func TestServer_ServesMetricsEndpointWithDefaultConfig(t *testing.T) {
	srv := baseserver.NewForTests(t)

	baseserver.StartServerForTests(t, srv)

	readyUR := fmt.Sprintf("%s/metrics", srv.HTTPAddress())
	resp, err := http.Get(readyUR)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestServer_ServesMetricsEndpointWithCustomMetricsConfig(t *testing.T) {
	registry := prometheus.NewRegistry()
	srv := baseserver.NewForTests(t,
		baseserver.WithMetricsRegistry(registry),
	)

	baseserver.StartServerForTests(t, srv)

	readyUR := fmt.Sprintf("%s/metrics", srv.HTTPAddress())
	resp, err := http.Get(readyUR)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestServer_ServesPprof(t *testing.T) {
	srv := baseserver.NewForTests(t)
	baseserver.StartServerForTests(t, srv)

	resp, err := http.Get(srv.HTTPAddress() + pprof.Path)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, resp.StatusCode, "must serve pprof on %s", pprof.Path)
}

func TestServer_WithTS(t *testing.T) {
	cert, key := getCertificatesAsFiles(t)
	srv := baseserver.NewForTests(t, baseserver.WithTLS(cert, key))

	baseserver.StartServerForTests(t, srv)
}

func getCertificatesAsFiles(t *testing.T) (string, string) {
	t.Helper()

	cert, err := ioutil.TempFile("", "localhost_cert")
	require.NoError(t, err)
	require.NoError(t, ioutil.WriteFile(cert.Name(), []byte(certtest.LocalhostPublic), 0644))
	t.Cleanup(func() {
		require.NoError(t, os.Remove(cert.Name()))
	})

	key, err := ioutil.TempFile("", "localhost_key")
	require.NoError(t, err)
	require.NoError(t, ioutil.WriteFile(key.Name(), []byte(certtest.LocalhostPrivate), 0644))
	t.Cleanup(func() {
		require.NoError(t, os.Remove(key.Name()))
	})

	return cert.Name(), key.Name()
}
