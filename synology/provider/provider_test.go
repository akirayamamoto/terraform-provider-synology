package provider

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	client "github.com/synology-community/go-synology"
	"github.com/synology-community/go-synology/pkg/api"
	"github.com/testcontainers/testcontainers-go"
	testcompose "github.com/testcontainers/testcontainers-go/modules/compose"
)

var providerFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"synology": providerserver.NewProtocol6WithError(New()()),
}

var testAccProtoV6ProviderFactories = providerFactories

func TestMain(m *testing.M) {
	if os.Getenv("TF_ACC") == "" {
		// short circuit non acceptance test runs
		os.Exit(m.Run())
	}

	os.Exit(runAcceptanceTests(m))
}

type logConsumer struct {
	StdOut bool
}

type dsmTestClient struct {
	host   string
	client *client.Client
}

func (l *logConsumer) Accept(logEntry testcontainers.Log) {
	if logEntry.LogType == testcontainers.StdoutLog && l.StdOut {
		fmt.Printf("[DSM] %s", logEntry.Content)
	}
	if logEntry.LogType == testcontainers.StderrLog {
		fmt.Printf("[DSM ERROR] %s", logEntry.Content)
	}
}

func runAcceptanceTests(m *testing.M) int {
	user := os.Getenv("TEST_SYNOLOGY_USER")
	if user == "" {
		user = "admin"
	}

	password := os.Getenv("TEST_SYNOLOGY_PASSWORD")
	externalHost := os.Getenv("TEST_SYNOLOGY_HOST")

	if err := os.Setenv("SYNOLOGY_USER", user); err != nil {
		panic(err)
	}

	if err := os.Setenv("SYNOLOGY_PASSWORD", password); err != nil {
		panic(err)
	}

	if err := os.Setenv("SYNOLOGY_SKIP_CERT_CHECK", "true"); err != nil {
		panic(err)
	}

	if externalHost != "" {
		cli, createErr := client.New(api.Options{
			Host:       externalHost,
			VerifyCert: false,
			RetryLimit: 5,
		})
		if createErr != nil {
			panic(createErr)
		}
		testClient, ok := cli.(*client.Client)
		if !ok {
			panic("failed to cast client")
		}
		readyHost, err := waitForDSMAPI(context.Background(), []dsmTestClient{{host: externalHost, client: testClient}}, user, password)
		if err != nil {
			panic(err)
		}
		if err := os.Setenv("SYNOLOGY_HOST", readyHost); err != nil {
			panic(err)
		}
		return m.Run()
	}

	// Disable Ryuk reaper to avoid connection issues in local development
	if err := os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true"); err != nil {
		panic(err)
	}

	dc, err := testcompose.NewDockerCompose("../../docker-compose.yaml")
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err = dc.WithOsEnv().Up(ctx, testcompose.Wait(true)); err != nil {
		panic(err)
	}

	defer func() {
		if err := dc.Down(
			context.Background(),
			testcompose.RemoveOrphans(true),
			testcompose.RemoveImagesLocal,
		); err != nil {
			panic(err)
		}
	}()

	container, err := dc.ServiceContainer(ctx, "dsm")
	if err != nil {
		panic(err)
	}

	lc := &logConsumer{StdOut: os.Getenv("DSM_STDOUT") != ""}

	testcontainers.WithLogConsumers(lc)

	endpoint, err := container.PortEndpoint(ctx, "5000/tcp", "http")
	if err != nil {
		panic(err)
	}

	// Discover both endpoints; we prefer HTTPS but can fall back to HTTP during boot.
	httpsEndpoint, err := container.PortEndpoint(ctx, "5001/tcp", "https")
	if err != nil {
		httpsEndpoint = endpoint
		fmt.Printf("Warning: HTTPS port not available, using HTTP: %s\n", endpoint)
	}

	cli, createErr := client.New(api.Options{
		Host:       httpsEndpoint,
		VerifyCert: false,
		RetryLimit: 5,
	})
	if createErr != nil {
		panic(createErr)
	}
	testClient, ok := cli.(*client.Client)
	if !ok {
		panic("failed to cast client")
	}
	clients := []dsmTestClient{{host: httpsEndpoint, client: testClient}}

	readyHost, err := waitForDSMAPI(ctx, clients, user, password)
	if err != nil {
		panic(err)
	}
	if err = os.Setenv("SYNOLOGY_HOST", readyHost); err != nil {
		panic(err)
	}

	return m.Run()
}

func preCheck(t *testing.T) {
	variables := []string{
		"SYNOLOGY_HOST",
		"SYNOLOGY_USER",
	}

	for _, variable := range variables {
		value := os.Getenv(variable)
		if value == "" {
			t.Fatalf("`%s` must be set for acceptance tests!", variable)
		}
	}
}

// waitForDSMAPI waits for the DSM API to be ready and accepting requests
// This is necessary because the container may report as healthy before the API is fully initialized.
func waitForDSMAPI(ctx context.Context, clients []dsmTestClient, user, password string) (string, error) {
	maxRetries := 120
	if runtime.GOARCH == "arm64" {
		maxRetries = 360
	}
	if configured := os.Getenv("TEST_DSM_MAX_RETRIES"); configured != "" {
		if parsed, err := strconv.Atoi(configured); err == nil && parsed > 0 {
			maxRetries = parsed
		}
	}
	retryDelay := 5 * time.Second

	fmt.Printf(
		"Waiting for DSM API to be ready (max %d attempts, %v between attempts)...\n",
		maxRetries,
		retryDelay,
	)

	var lastErr error
	for i := range maxRetries {
		for _, c := range clients {
			_, err := c.client.Login(ctx, api.LoginOptions{
				Username: user,
				Password: password,
			})
			if err == nil {
				fmt.Printf("✓ DSM API is ready after %d attempts via %s\n", i+1, c.host)
				return c.host, nil
			}
			lastErr = err
		}

		if i < maxRetries-1 {
			if (i+1)%10 == 0 {
				errText := ""
				if lastErr != nil {
					errText = lastErr.Error()
				}
				if strings.Contains(strings.ToLower(errText), "eof") {
					errText = fmt.Sprintf("%s (DSM may still be initializing TLS/auth)", errText)
				}
				fmt.Printf(
					"Still waiting... (attempt %d/%d): %v\n",
					i+1,
					maxRetries,
					errText,
				)
			}
			time.Sleep(retryDelay)
			continue
		}

		return "", fmt.Errorf(
			"DSM API did not become ready after %d attempts (waited %v): %w",
			maxRetries,
			time.Duration(maxRetries)*retryDelay,
			lastErr,
		)
	}

	return "", fmt.Errorf("DSM API did not become ready after %d attempts", maxRetries)
}
