// Copyright (c) 2022 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package main

import (
	"fmt"
	"github.com/gitpod-io/gitpod/common-go/baseserver"
	"github.com/gitpod-io/gitpod/common-go/log"
	"github.com/gitpod-io/gitpod/public-api-server/middleware"
	"github.com/gitpod-io/gitpod/public-api-server/pkg/apiv1"
	"github.com/gitpod-io/gitpod/public-api-server/pkg/proxy"
	v1 "github.com/gitpod-io/gitpod/public-api/v1"
	"net/http"
	"net/url"
)

func main() {
	logger := log.New()
	srv, err := baseserver.New("public_api_server",
		baseserver.WithLogger(logger),
		baseserver.WithHTTPPort(9000),
		baseserver.WithGRPCPort(9001),
	)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize public api server.")
	}

	if err := register(srv); err != nil {
		logger.WithError(err).Fatal("Failed to register services.")
	}

	if listenErr := srv.ListenAndServe(); listenErr != nil {
		logger.WithError(listenErr).Fatal("Failed to serve public api server")
	}
}

func register(srv *baseserver.Server) error {
	logger := log.New()
	m := middleware.NewLoggingMiddleware(logger)
	srv.HTTPMux().Handle("/", m(http.HandlerFunc(HelloWorldHandler)))

	// TODO(milan): Move to configuration
	serverAPI, err := url.Parse("wss://gitpod.io/api/v1")
	if err != nil {
		return fmt.Errorf("failed to parse server API: %w", err)
	}

	serverConnPool := &proxy.NoConnectionPool{ServerAPI: serverAPI}

	v1.RegisterWorkspacesServiceServer(srv.GRPC(), apiv1.NewWorkspaceService(serverConnPool))
	v1.RegisterPrebuildsServiceServer(srv.GRPC(), v1.UnimplementedPrebuildsServiceServer{})

	return nil
}

func HelloWorldHandler(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte(`hello world`))
}
