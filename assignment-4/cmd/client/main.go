package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"connectrpc.com/connect"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	v1 "github.com/stupside/DATA-intensive/assignment-4/modules/proto/gen/spec/v1"
	"github.com/stupside/DATA-intensive/assignment-4/modules/proto/gen/spec/v1/v1connect"
	"github.com/urfave/cli/v2"
	"golang.org/x/net/http2"
)

type clientConfig struct {
	ServerURL  string `koanf:"server_url"`
	CACertPath string `koanf:"ca_cert"`
	DeviceCert string `koanf:"device_cert"`
	DeviceKey  string `koanf:"device_key"`
}

var (
	cfg       *clientConfig
	outputDir string
)

func main() {
	var configPath string

	app := &cli.App{
		Name:    "client",
		Usage:   "Client for device onboarding and distributed file sharing",
		Version: "1.0.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Usage:       "Path to YAML configuration file",
				Destination: &configPath,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "output-dir",
				Usage:       "Base directory for downloaded files and keys",
				Destination: &outputDir,
				Required:    true,
			},
		},
		Before: func(c *cli.Context) error {
			k := koanf.New(".")
			if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			var loaded clientConfig
			if err := k.Unmarshal("", &loaded); err != nil {
				return fmt.Errorf("failed to unmarshal config: %w", err)
			}

			cfg = &loaded
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "device",
				Usage: "Manage device onboarding",
				Subcommands: []*cli.Command{
					{
						Name:  "onboard",
						Usage: "Onboard device and obtain device certificate",
						Action: func(c *cli.Context) error {
							return onboardCommand(c.Context)
						},
					},
				},
			},
			{
				Name:  "share",
				Usage: "Manage file shares",
				Subcommands: []*cli.Command{
					{
						Name:  "push",
						Usage: "Upload files and create a share",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "files",
								Usage:    "Regex pattern to match files",
								Required: true,
							},
						},
						Action: func(c *cli.Context) error {
							return pushCommand(c.Context, c.String("files"))
						},
					},
					{
						Name:  "pull",
						Usage: "Download files from a share",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "key",
								Usage:    "Share key to download from",
								Required: true,
							},
						},
						Action: func(c *cli.Context) error {
							return pullCommand(c.Context, c.String("key"))
						},
					},
					{
						Name:  "summary",
						Usage: "Get relay statistics for a share",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "key",
								Usage:    "Share key to get statistics for",
								Required: true,
							},
						},
						Action: func(c *cli.Context) error {
							return summaryCommand(c.Context, c.String("key"))
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

// =============================================================================
// ONBOARD COMMAND
// =============================================================================

func onboardCommand(ctx context.Context) error {
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %w", err)
	}

	slog.Info("Generating device key pair and CSR", "device_id", hostname)

	// Generate Ed25519 key pair
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Create CSR
	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: hostname,
		},
		SignatureAlgorithm: x509.PureEd25519,
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create CSR: %w", err)
	}
	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})

	// Save private key
	pkcs8, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8})
	keyPath := filepath.Join(outputDir, filepath.Base(cfg.DeviceKey))
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}
	slog.Info("Private key saved", "path", keyPath)

	// Create HTTP client with TLS-only (no client certificate for onboarding)
	httpClient, err := createHTTPClient(cfg.CACertPath, "", "")
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Create device service client
	deviceClient := v1connect.NewDeviceServiceClient(httpClient, cfg.ServerURL)

	slog.Info("Requesting device certificate from server")

	// Call Onboard endpoint
	resp, err := deviceClient.Onboard(ctx, connect.NewRequest(&v1.OnboardRequest{
		Csr: csrPEM,
	}))
	if err != nil {
		return fmt.Errorf("failed to onboard device: %w", err)
	}

	// Save certificate
	certPath := filepath.Join(outputDir, filepath.Base(cfg.DeviceCert))
	if err := os.WriteFile(certPath, resp.Msg.GetCertificate(), 0644); err != nil {
		return fmt.Errorf("failed to save certificate: %w", err)
	}

	slog.Info("Device certificate saved", "path", certPath)
	fmt.Printf("\nDevice onboarded successfully!\n")
	fmt.Printf("Certificate: %s\n", certPath)
	fmt.Printf("Private key: %s\n", keyPath)

	return nil
}

// =============================================================================
// PUSH COMMAND
// =============================================================================

func pushCommand(ctx context.Context, filesPattern string) error {
	fileRegex, err := regexp.Compile(filesPattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Create HTTP client with device certificate
	httpClient, err := createHTTPClient(cfg.CACertPath, cfg.DeviceCert, cfg.DeviceKey)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	shareClient := v1connect.NewShareServiceClient(httpClient, cfg.ServerURL)
	relayClient := v1connect.NewRelayServiceClient(httpClient, cfg.ServerURL)

	slog.Info("Scanning files", "pattern", filesPattern)

	// Find matching files
	var files []struct {
		path string
		size uint64
	}

	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if fileRegex.MatchString(path) || fileRegex.MatchString(info.Name()) {
			files = append(files, struct {
				path string
				size uint64
			}{
				path: path,
				size: uint64(info.Size()),
			})
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to find files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files matched the pattern")
	}

	slog.Info("Found files", "count", len(files))

	// Create share
	shareReq := &v1.CreateRequest{
		Contents: make([]*v1.CreateRequest_Content, len(files)),
	}
	for i, file := range files {
		shareReq.Contents[i] = &v1.CreateRequest_Content{
			Size: file.size,
			Path: file.path,
		}
	}

	shareResp, err := shareClient.Create(ctx, connect.NewRequest(shareReq))
	if err != nil {
		return fmt.Errorf("failed to create share: %w", err)
	}

	shareKey := shareResp.Msg.GetKey()
	shareID := shareResp.Msg.GetId()
	slog.Info("Share created", "id", shareID, "key", shareKey)

	// Start streaming with bidirectional communication
	slog.Info("Starting file upload")
	stream := relayClient.Stream(ctx)

	// Send StreamInit first
	if err := stream.Send(&v1.StreamRequest{
		Payload: &v1.StreamRequest_Init{
			Init: &v1.StreamInit{
				ShareKey: shareKey,
			},
		},
	}); err != nil {
		return fmt.Errorf("failed to send init: %w", err)
	}

	// Open all files for reading
	openFiles := make(map[string]*os.File)
	defer func() {
		for _, file := range openFiles {
			if file != nil {
				file.Close()
			}
		}
	}()

	for _, fileInfo := range files {
		file, err := os.Open(fileInfo.path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", fileInfo.path, err)
		}
		openFiles[fileInfo.path] = file
	}

	// Handle server chunk requests
	for {
		msg, err := stream.Receive()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("stream receive error: %w", err)
		}

		if req := msg.GetRequest(); req != nil {
			// Server is requesting a chunk
			path := req.GetPath()
			offset := req.GetOffset()
			length := req.GetLength()

			file, exists := openFiles[path]
			if !exists {
				return fmt.Errorf("server requested unknown file: %s", path)
			}

			// Seek to the requested offset
			if _, err := file.Seek(int64(offset), io.SeekStart); err != nil {
				return fmt.Errorf("failed to seek file %s: %w", path, err)
			}

			// Read the requested chunk
			buffer := make([]byte, length)
			n, readErr := io.ReadFull(file, buffer)
			if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
				return fmt.Errorf("failed to read file %s: %w", path, readErr)
			}

			// Send the chunk using StreamChunk
			if err := stream.Send(&v1.StreamRequest{
				Payload: &v1.StreamRequest_Chunk{
					Chunk: &v1.StreamChunk{
						Data:   buffer[:n],
						Path:   path,
						Offset: offset,
						Length: uint64(n),
					},
				},
			}); err != nil {
				return fmt.Errorf("failed to send chunk: %w", err)
			}

			slog.Info("Sent chunk", "path", path, "offset", offset, "length", n)

		} else if msg.GetComplete() != nil {
			// Server indicates completion
			slog.Info("Upload complete - server signaled completion")
			break
		}
	}

	// Close stream
	if err := stream.CloseRequest(); err != nil {
		return fmt.Errorf("failed to close stream: %w", err)
	}

	fmt.Printf("\nShare created successfully!\n")
	fmt.Printf("Share ID: %s\n", shareID)
	fmt.Printf("Share Key: %s\n", shareKey)
	fmt.Printf("\nTo download, run: %s share pull --key %s\n", os.Args[0], shareKey)

	return nil
}

// =============================================================================
// PULL COMMAND
// =============================================================================

func pullCommand(ctx context.Context, shareKey string) error {
	// Create HTTP client with device certificate
	httpClient, err := createHTTPClient(cfg.CACertPath, cfg.DeviceCert, cfg.DeviceKey)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	shareClient := v1connect.NewShareServiceClient(httpClient, cfg.ServerURL)
	relayClient := v1connect.NewRelayServiceClient(httpClient, cfg.ServerURL)

	slog.Info("Getting share details", "share_key", shareKey)

	// Parse share key as numeric ID for Detail endpoint
	id, err := strconv.ParseUint(shareKey, 10, 64)
	if err != nil {
		// If not a number, try it as-is (might be the actual key)
		slog.Warn("Share key is not numeric, might cause issues", "key", shareKey)
		id = 0
	}

	// Get share details to know what files to download (if we have a valid ID)
	var files []*v1.DetailResponse_Content
	if id > 0 {
		detailResp, err := shareClient.Detail(ctx, connect.NewRequest(&v1.DetailRequest{
			Id: id,
		}))
		if err != nil {
			slog.Warn("Failed to get share details, will attempt download anyway", "error", err)
		} else {
			files = detailResp.Msg.GetContents()
			slog.Info("Share contains files", "file_count", len(files))
		}
	}

	slog.Info("Starting download", "share_key", shareKey)

	// Start bidirectional consume stream
	stream := relayClient.Consume(ctx)

	// Send ConsumeInit with share key
	if err := stream.Send(&v1.ConsumeRequest{
		Payload: &v1.ConsumeRequest_Init{
			Init: &v1.ConsumeInit{
				ShareKey: shareKey,
			},
		},
	}); err != nil {
		return fmt.Errorf("failed to send init: %w", err)
	}

	// Track open files
	openFiles := make(map[string]*os.File)
	defer func() {
		for _, file := range openFiles {
			if file != nil {
				file.Close()
			}
		}
	}()

	// Server drives the download - just receive chunks
	totalBytesReceived := uint64(0)
	filesReceived := 0

	for {
		msg, err := stream.Receive()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("stream receive error: %w", err)
		}

		if chunk := msg.GetChunk(); chunk != nil {
			path := chunk.GetPath()
			offset := chunk.GetOffset()
			data := chunk.GetData()

			// Open file if not already open
			filePath := filepath.Join(outputDir, path)
			file, exists := openFiles[filePath]
			if !exists {
				// Create parent directories
				dir := filepath.Dir(filePath)
				if err := os.MkdirAll(dir, 0755); err != nil {
					return fmt.Errorf("failed to create directory %s: %w", dir, err)
				}

				// Create file
				file, err = os.Create(filePath)
				if err != nil {
					return fmt.Errorf("failed to create file %s: %w", path, err)
				}
				openFiles[filePath] = file
				filesReceived++
				slog.Info("Started downloading file", "path", path, "file_number", filesReceived)
			}

			// Write chunk data at offset
			if len(data) > 0 {
				if _, err := file.WriteAt(data, int64(offset)); err != nil {
					return fmt.Errorf("failed to write to file %s: %w", filePath, err)
				}
			}

			totalBytesReceived += uint64(len(data))
			slog.Debug("Received chunk", "path", path, "offset", offset, "length", len(data))

		} else if msg.GetComplete() != nil {
			// Server indicates completion
			slog.Info("Download complete - server signaled completion")
			break
		}
	}

	// Close all files
	for path, file := range openFiles {
		if file != nil {
			file.Close()
			openFiles[path] = nil
			slog.Info("File download complete", "path", path)
		}
	}

	// Close stream
	if err := stream.CloseRequest(); err != nil {
		return fmt.Errorf("failed to close stream: %w", err)
	}

	fmt.Printf("\nDownload complete!\n")
	fmt.Printf("Files received: %d\n", filesReceived)
	fmt.Printf("Total bytes: %d\n", totalBytesReceived)

	return nil
}

// =============================================================================
// SUMMARY COMMAND
// =============================================================================

func summaryCommand(ctx context.Context, shareKey string) error {
	// Create HTTP client with device certificate
	httpClient, err := createHTTPClient(cfg.CACertPath, cfg.DeviceCert, cfg.DeviceKey)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	relayClient := v1connect.NewRelayServiceClient(httpClient, cfg.ServerURL)

	slog.Info("Getting relay statistics", "share_key", shareKey)

	// Call Summary endpoint
	resp, err := relayClient.Summary(ctx, connect.NewRequest(&v1.SummaryRequest{
		ShareKey: shareKey,
	}))
	if err != nil {
		return fmt.Errorf("failed to get summary: %w", err)
	}

	files := resp.Msg.GetFiles()
	if len(files) == 0 {
		fmt.Println("\nNo files found for this share.")
		return nil
	}

	fmt.Printf("\nRelay Statistics for Share: %s\n", shareKey)
	fmt.Println("========================================")

	var totalRelayed uint64
	var totalRemaining uint64

	for i, file := range files {
		fmt.Printf("\nFile %d: %s\n", i+1, file.GetPath())
		fmt.Printf("  Bytes Relayed:   %d\n", file.GetBytesRelayed())
		fmt.Printf("  Bytes Remaining: %d\n", file.GetBytesRemaining())

		totalSize := file.GetBytesRelayed() + file.GetBytesRemaining()
		if totalSize > 0 {
			percentage := float64(file.GetBytesRelayed()) / float64(totalSize) * 100
			fmt.Printf("  Progress:        %.2f%%\n", percentage)
		}

		totalRelayed += file.GetBytesRelayed()
		totalRemaining += file.GetBytesRemaining()
	}

	fmt.Println("\n========================================")
	fmt.Printf("Total Files:       %d\n", len(files))
	fmt.Printf("Total Relayed:     %d bytes\n", totalRelayed)
	fmt.Printf("Total Remaining:   %d bytes\n", totalRemaining)

	grandTotal := totalRelayed + totalRemaining
	if grandTotal > 0 {
		percentage := float64(totalRelayed) / float64(grandTotal) * 100
		fmt.Printf("Overall Progress:  %.2f%%\n", percentage)
	}

	return nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func createHTTPClient(caPath, certPath, keyPath string) (*http.Client, error) {
	// Load CA certificate
	caCert, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	tlsConfig := &tls.Config{
		RootCAs:    caCertPool,
		MinVersion: tls.VersionTLS12,
		NextProtos: []string{"h2"}, // Enable HTTP/2 via ALPN
	}

	// Load client certificate if paths are provided
	if certPath != "" && keyPath != "" {
		clientCert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}

	// Create HTTP/2 transport
	transport := &http2.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &http.Client{
		Transport: transport,
	}, nil
}
