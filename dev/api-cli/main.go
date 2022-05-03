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
	"google.golang.org/protobuf/encoding/protojson"
	"net/http"
	"os"
	"strings"
)

var (
	apiAddress string
)

func main() {
	log.Init("api-cli", "", false, true)
	logger := log.Log
	ctx := context.Background()
	cmd := &cobra.Command{
		Use: "api-cli",
	}

	cmd.PersistentFlags().StringVar(&apiAddress, "address", "api.main.preview.gitpod-dev.com:443", "Address of the API endpoint. Should be in the form <host>:<port>.")

	cmd.AddCommand(newWorkspaceCommand())

	if err := cmd.ExecuteContext(ctx); err != nil {
		logger.WithError(err).Fatal("Failed to run command.")
	}
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
			workspace, err := getWorkspace(cmd.Context(), apiAddress)
			log.Log.WithError(err).WithField("workspace", workspace.String()).Debugf("Workspace response")
			if err != nil {
				log.Log.WithError(err).Fatal("Failed to retrieve workspace.")
				return
			}

			data, err := protojson.Marshal(workspace)
			if err != nil {
				log.Log.WithError(err).Fatal("Failed to serialize workspace into json")
			}
			_, _ = fmt.Fprint(os.Stdout, string(data))
		},
	}

	return cmd
}

func getWorkspace(ctx context.Context, address string) (*v1.GetWorkspaceResponse, error) {
	conn, err := newConn(address)
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

func newConn(address string) (*grpc.ClientConn, error) {
	// Firstly, we need to obtain the server public certificate, we can do that by issuing a regular https request and then
	// inspecting the cert
	// For now, we just strip off the `api.` part to hit the site that's actually serving HTTPS traffic
	certURL := fmt.Sprintf("https://%s", strings.ReplaceAll(address, "api.", ""))
	log.Log.Debugf("Retrieving public certificate chain from URL: %s", certURL)

	cert, err := getServerCertificateChain(certURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get certs: %w", err)
	}

	transport, err := tlsTransport(cert)
	if err != nil {
		return nil, fmt.Errorf("failed to construct tls transport")
	}

	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(transport))
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", address, err)
	}

	return conn, nil
}

func getServerCertificateChain(url string) ([]byte, error) {
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
