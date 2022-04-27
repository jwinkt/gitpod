package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/gitpod-io/gitpod/common-go/log"
	v1 "github.com/gitpod-io/gitpod/public-api/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net/http"
	"strings"
)

func main() {
	log.Init("api-cli", "", false, true)
	logger := log.Log
	ctx := context.Background()
	cmd := &cobra.Command{
		Use: "api-cli",
	}

	cmd.AddCommand(newWorkspaceCommand())

	if err := cmd.ExecuteContext(ctx); err != nil {
		logger.WithError(err).Fatal("Failed to run command.")
	}
	logger.Info("Command completed.")
}

func newWorkspaceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "workspace",
	}

	cmd.AddCommand(newWorkspaceGetCommand())

	return cmd
}

func newWorkspaceGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "get",
		Run: func(cmd *cobra.Command, args []string) {
			log.Log.Info("Running get command")
			workspace, err := getWorkspace(cmd.Context(), "api.mp-papi-caddy-grpc.preview.gitpod-dev.com:443")
			if err != nil {
				log.Log.WithError(err).Errorf("Failed to retrieve workspace.")
			}

			log.Log.Infof("Got workspace: %v", workspace.String())
		},
	}

	return cmd
}

func getWorkspace(ctx context.Context, url string) (*v1.GetWorkspaceResponse, error) {
	conn, err := newConn(url)
	if err != nil {
		return nil, fmt.Errorf("failed to create new connection: %w", err)
	}

	workspace := v1.NewWorkspacesServiceClient(conn)

	resp, err := workspace.GetWorkspace(ctx, &v1.GetWorkspaceRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve workspace: %w", err)
	}

	return resp, nil
}

func newConn(url string) (*grpc.ClientConn, error) {
	// For now, we just strip off the `api.` part to hit the site that's actually serving HTTPS traffic
	certURL := strings.ReplaceAll(url, "api.", "")
	cert, err := getCrets(fmt.Sprintf("https://%s", certURL))
	if err != nil {
		return nil, fmt.Errorf("failed to get certs: %w", err)
	}

	transport, err := tlsTransport(cert)
	if err != nil {
		return nil, fmt.Errorf("failed to construct tls transport")
	}

	conn, err := grpc.Dial(url, grpc.WithTransportCredentials(transport))
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", url, err)
	}

	return conn, nil
}

func getCrets(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	var pemBytes bytes.Buffer
	for _, cert := range resp.TLS.PeerCertificates {
		if err := pem.Encode(&pemBytes, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}); err != nil {
			return nil, fmt.Errorf("failed to encode certificates: %w", err)
		}
	}

	return pemBytes.Bytes(), nil
}

func tlsTransport(certs []byte) (credentials.TransportCredentials, error) {
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certs) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	// Create the credentials and return it
	config := &tls.Config{
		RootCAs: certPool,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		MinVersion:       tls.VersionTLS12,
		MaxVersion:       tls.VersionTLS12,
		NextProtos:       []string{"h2"},
	}

	return credentials.NewTLS(config), nil
}
