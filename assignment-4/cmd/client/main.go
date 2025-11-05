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
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pterm/pterm"

	"connectrpc.com/connect"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	v1 "github.com/stupside/DATA-intensive/assignment-4/proto/gen/spec/v1"
	"github.com/stupside/DATA-intensive/assignment-4/proto/gen/spec/v1/v1connect"
	"github.com/urfave/cli/v2"
	"golang.org/x/net/http2"
)

// ============================================================================
// CONSTANTS - All hardcoded strings extracted here
// ============================================================================

const (
	// Messages
	msgUsingConfig           = "Using config: %s"
	msgConfigFailed          = "Failed to load config: %v"
	msgGeneratingKeys        = "Generating device key pair and CSR (device_id: %s)"
	msgPrivateKeySaved       = "Private key saved: %s"
	msgRequestingCert        = "Requesting device certificate from server"
	msgCertSaved             = "Device certificate saved: %s"
	msgOnboardSuccess        = "Device onboarded successfully!"
	msgPrivateKeyLabel       = "Private key: %s"
	msgCertificateLabel      = "Certificate: %s"
	msgFetchingConnections   = "Fetching connection history..."
	msgNoConnections         = "No connections found"
	msgFetchingShares        = "Fetching shares created by this device..."
	msgNoShares              = "No shares found"
	msgScanningSinglePattern = "Scanning files with pattern: %s"
	msgScanningMultiPattern  = "Scanning files with %d patterns"
	msgFilesFound            = "Found %d files"
	msgNoFilesMatched        = "no files matched the pattern"
	msgShareCreated          = "Share created (id: %d, share_secret: %s)"
	msgStartingUpload        = "Starting file upload"
	msgUploadComplete        = "Upload complete - server signaled completion"
	msgShareCreatedSuccess   = "Share created successfully!"
	msgShareIDInfo           = "Share ID: %d"
	msgShareSecretInfo       = "Share Secret: %s"
	msgDownloadCommand       = "To download, run: %s share pull --id %d --share-secret %s"
	msgGettingShareDetails   = "Getting share details (share_secret: %s)"
	msgShareDetailsFailed    = "Failed to get share details, will attempt download anyway: %v"
	msgShareContainsFiles    = "Share contains %d files"
	msgStartingDownload      = "Starting download for share %s"
	msgFileDownloadStarted   = "Started downloading file: %s (file #%d)"
	msgDownloadComplete      = "Download complete - server signaled completion"
	msgFileComplete          = "File download complete: %s"
	msgDownloadSuccess       = "Download complete!"
	msgFilesReceived         = "Files received: %d"
	msgTotalBytes            = "Total bytes: %d"
	msgGettingStats          = "Getting relay statistics (share_id: %d, share_secret: %s)"
	msgNoShareFiles          = "No files found for this share."
	msgApplicationError      = "application error: %v"

	// Table Headers
	hdrConnectionID     = "ID"
	hdrConnectionTime   = "TIMESTAMP"
	hdrConnectionIP     = "IP ADDRESS"
	hdrConnectionStatus = "STATUS"

	hdrShareID      = "SHARE ID"
	hdrShareFiles   = "FILE COUNT"
	hdrShareSecret  = "SHARE SECRET"
	hdrShareCreated = "CREATED AT"

	hdrFile      = "File"
	hdrRelayed   = "Relayed"
	hdrRemaining = "Remaining"
	hdrProgress  = "Progress"

	// Labels
	lblBytesRelayed    = "Bytes Relayed"
	lblBytesRemaining  = "Bytes Remaining"
	lblProgress        = "Progress"
	lblTotalFiles      = "Total Files"
	lblTotalRelayed    = "Total Relayed"
	lblTotalRemaining  = "Total Remaining"
	lblOverallProgress = "Overall Progress"

	// Format strings
	fmtTimestamp = "2006-01-02 15:04:05"
	fmtPercent   = "%.2f%%"

	// Status values
	statusSuccess = "SUCCESS"
	statusFailed  = "FAILED"

	// Section titles
	titleRelayStats     = "Relay Statistics for Share: %s (id=%d)"
	titleOverallSummary = "Overall Summary"
)

// ============================================================================
// TYPES - Config struct
// ============================================================================

type clientConfig struct {
	ServerURL  string `koanf:"server_url"`
	CACertPath string `koanf:"ca_cert"`
	DeviceCert string `koanf:"device_cert"`
	DeviceKey  string `koanf:"device_key"`
}

// Clients holds all gRPC service clients
type Clients struct {
	HTTP   *http.Client
	Auth   v1connect.AuthServiceClient
	Device v1connect.DeviceServiceClient
	Share  v1connect.ShareServiceClient
	Relay  v1connect.RelayServiceClient
}

var (
	cfg       *clientConfig
	outputDir string
)

func loadConfig(configPath string) error {
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
}

// loadConfigOrDie loads config or exits on error
func loadConfigOrDie(configPath string) {
	pterm.Info.Printfln(msgUsingConfig, configPath)
	if err := loadConfig(configPath); err != nil {
		pterm.Error.Printfln(msgConfigFailed, err)
		os.Exit(1)
	}
}

// createClientsOrDie creates all clients or exits on error
func createClientsOrDie(needsCert bool) *Clients {
	certPath, keyPath := "", ""
	if needsCert {
		certPath, keyPath = cfg.DeviceCert, cfg.DeviceKey
	}

	httpClient, err := createHTTPClient(cfg.CACertPath, certPath, keyPath)
	if err != nil {
		pterm.Error.Printfln("Failed to create HTTP client: %v", err)
		os.Exit(1)
	}

	return &Clients{
		HTTP:   httpClient,
		Auth:   v1connect.NewAuthServiceClient(httpClient, cfg.ServerURL),
		Device: v1connect.NewDeviceServiceClient(httpClient, cfg.ServerURL),
		Share:  v1connect.NewShareServiceClient(httpClient, cfg.ServerURL),
		Relay:  v1connect.NewRelayServiceClient(httpClient, cfg.ServerURL),
	}
}

// formatStatus formats boolean status as SUCCESS or FAILED
func formatStatus(success bool) string {
	if success {
		return statusSuccess
	}
	return statusFailed
}

// ============================================================================
// FILE AND SHARE HELPERS
// ============================================================================

// globFilesOrDie expands file patterns and returns file list with sizes
func globFilesOrDie(filesPatterns []string) []struct {
	path string
	size uint64
} {
	// Collect all matching files from patterns and direct paths
	matchMap := make(map[string]bool) // Use map to deduplicate

	for _, pattern := range filesPatterns {
		// Strip leading "./" from pattern if present
		pattern = strings.TrimPrefix(pattern, "./")

		// Check if this looks like a glob pattern (contains wildcards)
		if strings.ContainsAny(pattern, "*?[") {
			// It's a glob pattern - expand it
			matches, err := doublestar.Glob(os.DirFS("."), pattern)
			if err != nil {
				pterm.Error.Printfln("Invalid glob pattern '%s': %v", pattern, err)
				os.Exit(1)
			}
			for _, match := range matches {
				matchMap[match] = true
			}
		} else {
			// It's a direct file path - check if it exists
			if info, err := os.Stat(pattern); err == nil && !info.IsDir() {
				matchMap[pattern] = true
			}
		}
	}

	// Convert map to slice
	matches := make([]string, 0, len(matchMap))
	for path := range matchMap {
		matches = append(matches, path)
	}

	if len(filesPatterns) == 1 {
		pterm.Info.Printfln(msgScanningSinglePattern, filesPatterns[0])
	} else {
		pterm.Info.Printfln(msgScanningMultiPattern, len(filesPatterns))
	}

	// Find matching files and get their sizes
	var files []struct {
		path string
		size uint64
	}
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		files = append(files, struct {
			path string
			size uint64
		}{
			path: path,
			size: uint64(info.Size()),
		})
	}

	if len(files) == 0 {
		pterm.Error.Println(msgNoFilesMatched)
		os.Exit(1)
	}

	pterm.Success.Printfln(msgFilesFound, len(files))
	return files
}

// createShareOrDie creates a share via API or exits on error
func createShareOrDie(ctx context.Context, shareClient v1connect.ShareServiceClient, files []struct {
	path string
	size uint64
}) (shareID uint64, shareSecret string) {
	// Create share
	shareReq := &v1.CreateRequest{
		Files: make([]*v1.CreateRequest_File, len(files)),
	}
	for i, file := range files {
		shareReq.Files[i] = &v1.CreateRequest_File{
			Size: file.size,
			Path: file.path,
		}
	}

	shareResp, err := shareClient.Create(ctx, connect.NewRequest(shareReq))
	if err != nil {
		pterm.Error.Printfln("Failed to create share: %v", err)
		os.Exit(1)
	}

	shareSecret = shareResp.Msg.GetShareSecret()
	shareID = shareResp.Msg.GetId()
	pterm.Success.Printfln(msgShareCreated, shareID, shareSecret)

	return shareID, shareSecret
}

// ============================================================================
// HELPER FUNCTIONS - Reusable utilities
// ============================================================================

// setupCommand loads config and creates HTTP client (eliminates duplication)
func setupCommand(c *cli.Context, needsCert bool) (*http.Client, error) {
	cfgPath := c.String("config")
	pterm.Info.Printfln(msgUsingConfig, cfgPath)

	if err := loadConfig(cfgPath); err != nil {
		pterm.Error.Printfln(msgConfigFailed, err)
		return nil, err
	}

	// Create HTTP client
	certPath, keyPath := "", ""
	if needsCert {
		certPath, keyPath = cfg.DeviceCert, cfg.DeviceKey
	}

	return createHTTPClient(cfg.CACertPath, certPath, keyPath)
}

// ============================================================================
// TABLE RENDERING - pterm.Table implementations
// ============================================================================

// renderConnectionsTable displays connection history in a professional table
func renderConnectionsTable(connections []*v1.ConnectionInfo) error {
	if len(connections) == 0 {
		pterm.Info.Println(msgNoConnections)
		return nil
	}

	data := pterm.TableData{
		{hdrConnectionID, hdrConnectionTime, hdrConnectionIP, hdrConnectionStatus},
	}

	for _, conn := range connections {
		data = append(data, []string{
			fmt.Sprintf("%d", conn.Id),
			conn.GetCreatedAt().AsTime().Format(fmtTimestamp),
			conn.IpAddress,
			formatStatus(conn.Success),
		})
	}

	if err := pterm.DefaultTable.WithHasHeader().WithData(data).Render(); err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	pterm.DefaultSection.Printfln("Total connections: %d", len(connections))
	return nil
}

// renderSharesTable displays shares created by this device in a professional table
func renderSharesTable(shares []*v1.ListResponse_ShareInfo) error {
	if len(shares) == 0 {
		pterm.Info.Println(msgNoShares)
		return nil
	}

	data := pterm.TableData{
		{hdrShareID, hdrShareFiles, hdrShareSecret, hdrShareCreated},
	}

	for _, share := range shares {
		timestamp := time.Unix(
			int64(share.CreatedAt.GetSeconds()),
			int64(share.CreatedAt.GetNanos()),
		).Format(fmtTimestamp)

		data = append(data, []string{
			fmt.Sprintf("%d", share.Id),
			fmt.Sprintf("%d", share.FileCount),
			share.ShareSecret,
			timestamp,
		})
	}

	if err := pterm.DefaultTable.WithHasHeader().WithData(data).Render(); err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	pterm.DefaultSection.Printfln("Total shares: %d", len(shares))
	return nil
}

// renderShareSummary displays relay statistics with table and summary box
func renderShareSummary(shareID uint64, shareSecret string, files []*v1.SummaryResponse_FileSummary) {
	pterm.DefaultSection.Printfln(titleRelayStats, shareSecret, shareID)

	// File details table
	tableData := pterm.TableData{
		{hdrFile, hdrRelayed, hdrRemaining, hdrProgress},
	}

	var totalRelayed, totalRemaining uint64
	for _, file := range files {
		relayed := file.GetBytesRelayed()
		remaining := file.GetBytesRemaining()
		total := relayed + remaining

		progress := "0.00%"
		if total > 0 {
			progress = fmt.Sprintf(fmtPercent, float64(relayed)/float64(total)*100)
		}

		tableData = append(tableData, []string{
			file.GetPath(),
			fmt.Sprintf("%d", relayed),
			fmt.Sprintf("%d", remaining),
			progress,
		})

		totalRelayed += relayed
		totalRemaining += remaining
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()

	// Overall summary in a box
	grandTotal := totalRelayed + totalRemaining
	overallProgress := "0.00%"
	if grandTotal > 0 {
		overallProgress = fmt.Sprintf(fmtPercent, float64(totalRelayed)/float64(grandTotal)*100)
	}

	pterm.DefaultBox.WithTitle(titleOverallSummary).Printf(
		"%s: %d\n%s: %d bytes\n%s: %d bytes\n%s: %s",
		lblTotalFiles, len(files),
		lblTotalRelayed, totalRelayed,
		lblTotalRemaining, totalRemaining,
		lblOverallProgress, overallProgress,
	)
}

// ============================================================================
// MAIN & CLI SETUP
// ============================================================================

func main() {
	app := &cli.App{
		Name:    "client",
		Usage:   "Client for device onboarding and distributed file sharing",
		Version: "1.0.0",
		Commands: []*cli.Command{
			newDeviceCommand(),
			newShareCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		pterm.Error.Printfln(msgApplicationError, err)
		os.Exit(1)
	}
}

// -------------------------
// Command builders & flags
// -------------------------

func configFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:  "config",
		Usage: "Path to YAML configuration file",
		Value: "./config.client.yml",
	}
}

func outputDirFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:  "output-dir",
		Usage: "Directory to store downloaded files or keys",
	}
}

func filesFlag() *cli.StringSliceFlag {
	return &cli.StringSliceFlag{
		Name:     "files",
		Usage:    "Glob pattern(s) or file paths (e.g. '**/*.go' or multiple files). Quote patterns to prevent shell expansion.",
		Required: true,
	}
}

func keyFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:     "share-secret",
		Usage:    "Share secret to download from (pair with share ID)",
		Required: true,
	}
}

func idFlag() *cli.Int64Flag {
	// Use an Int64Flag so the CLI requires a numeric value for share IDs.
	// We'll cast to uint64 when calling RPCs which expect unsigned IDs.
	return &cli.Int64Flag{
		Name:     "id",
		Usage:    "Numeric share ID to identify the share",
		Required: true,
	}
}

func newDeviceCommand() *cli.Command {
	return &cli.Command{
		Name:  "device",
		Usage: "Manage device onboarding",
		Subcommands: []*cli.Command{
			{
				Name:  "onboard",
				Usage: "Onboard device and obtain device certificate",
				Flags: []cli.Flag{configFlag()},
				Action: func(c *cli.Context) error {
					if err := deviceOnboardAction(c); err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:  "connections",
				Usage: "View connection history for this device",
				Flags: []cli.Flag{configFlag()},
				Action: func(c *cli.Context) error {
					return deviceConnectionsAction(c)
				},
			},
			{
				Name:  "shares",
				Usage: "List all shares created by this device",
				Flags: []cli.Flag{configFlag()},
				Action: func(c *cli.Context) error {
					return deviceSharesAction(c)
				},
			},
		},
	}
}

func newShareCommand() *cli.Command {
	return &cli.Command{
		Name:  "share",
		Usage: "Manage file shares",
		Subcommands: []*cli.Command{
			{
				Name:  "push",
				Usage: "Upload files and create a share",
				Flags: []cli.Flag{configFlag(), filesFlag()},
				Action: func(c *cli.Context) error {
					return sharePushAction(c)
				},
			},
			{
				Name:  "pull",
				Usage: "Download files from a share",
				Flags: []cli.Flag{configFlag(), idFlag(), keyFlag(), outputDirFlag()},
				Action: func(c *cli.Context) error {
					return sharePullAction(c)
				},
			},
			{
				Name:  "summary",
				Usage: "Get relay statistics for a share",
				Flags: []cli.Flag{configFlag(), idFlag(), keyFlag()},
				Action: func(c *cli.Context) error {
					return shareSummaryAction(c)
				},
			},
		},
	}
}

// -------------------------
// Command actions
// -------------------------

func deviceOnboardAction(c *cli.Context) error {
	_, err := setupCommand(c, false) // false = no client cert needed for onboarding
	if err != nil {
		return err
	}
	return onboardCommand(c.Context)
}

func sharePushAction(c *cli.Context) error {
	_, err := setupCommand(c, true) // true = needs client cert
	if err != nil {
		return err
	}
	return pushCommand(c.Context, c.StringSlice("files"))
}

func sharePullAction(c *cli.Context) error {
	_, err := setupCommand(c, true) // true = needs client cert
	if err != nil {
		return err
	}
	outputDir = c.String("output-dir")
	id := uint64(c.Int64("id"))
	return pullCommand(c.Context, id, c.String("share-secret"))
}

func shareSummaryAction(c *cli.Context) error {
	_, err := setupCommand(c, true) // true = needs client cert
	if err != nil {
		return err
	}
	id := uint64(c.Int64("id"))
	return summaryCommand(c.Context, id, c.String("share-secret"))
}

func deviceConnectionsAction(c *cli.Context) error {
	_, err := setupCommand(c, true) // true = needs client cert
	if err != nil {
		return err
	}
	return connectionsCommand(c.Context)
}

func deviceSharesAction(c *cli.Context) error {
	_, err := setupCommand(c, true) // true = needs client cert
	if err != nil {
		return err
	}
	return sharesCommand(c.Context)
}

// ============================================================================
// COMMAND IMPLEMENTATIONS
// ============================================================================

// =============================================================================
// ONBOARD COMMAND
// =============================================================================

func onboardCommand(ctx context.Context) error {
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %w", err)
	}

	pterm.Info.Printfln(msgGeneratingKeys, hostname)

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

	// Use directory from config's device_key path
	keyDir := filepath.Dir(cfg.DeviceKey)
	if err := os.MkdirAll(keyDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(cfg.DeviceKey, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}
	pterm.Success.Printfln(msgPrivateKeySaved, cfg.DeviceKey)

	// Create clients (no client certificate needed for onboarding)
	clients := createClientsOrDie(false)

	pterm.Info.Println(msgRequestingCert)

	// Call Onboard endpoint
	resp, err := clients.Auth.Onboard(ctx, connect.NewRequest(&v1.OnboardRequest{
		Csr: csrPEM,
	}))
	if err != nil {
		return fmt.Errorf("failed to onboard device: %w", err)
	}

	// Save certificate
	if err := os.WriteFile(cfg.DeviceCert, resp.Msg.GetCertificate(), 0644); err != nil {
		return fmt.Errorf("failed to save certificate: %w", err)
	}

	pterm.Success.Printfln(msgCertSaved, cfg.DeviceCert)
	pterm.Success.Println(msgOnboardSuccess)
	pterm.Info.Printfln(msgPrivateKeyLabel, cfg.DeviceKey)
	pterm.Info.Printfln(msgCertificateLabel, cfg.DeviceCert)

	return nil
}

// =============================================================================
// CONNECTIONS COMMAND
// =============================================================================

func connectionsCommand(ctx context.Context) error {
	// Create clients with device certificate
	clients := createClientsOrDie(true)

	pterm.Info.Println(msgFetchingConnections)

	// Call Connections RPC
	resp, err := clients.Device.Connections(ctx, connect.NewRequest(&v1.ConnectionsRequest{}))
	if err != nil {
		return fmt.Errorf("failed to fetch connections: %w", err)
	}

	// Output results as table
	return renderConnectionsTable(resp.Msg.Connections)
}

// =============================================================================
// SHARES COMMAND
// =============================================================================

func sharesCommand(ctx context.Context) error {
	// Create clients with device certificate
	clients := createClientsOrDie(true)

	pterm.Info.Println(msgFetchingShares)

	// Call List RPC endpoint
	resp, err := clients.Share.List(ctx, connect.NewRequest(&v1.ListRequest{}))
	if err != nil {
		return fmt.Errorf("failed to fetch shares: %w", err)
	}

	return renderSharesTable(resp.Msg.Shares)
}

// =============================================================================
// PUSH COMMAND
// =============================================================================

func pushCommand(ctx context.Context, filesPatterns []string) error {
	// Glob files and get their sizes
	files := globFilesOrDie(filesPatterns)

	// Create clients
	clients := createClientsOrDie(true)

	// Create share via API
	shareID, shareSecret := createShareOrDie(ctx, clients.Share, files)

	// Calculate total size for progress bar
	var totalSize int64
	for _, f := range files {
		totalSize += int64(f.size)
	}

	// Start streaming with bidirectional communication
	pterm.Info.Println(msgStartingUpload)

	// Create progress bar
	progressBar, _ := pterm.DefaultProgressbar.
		WithTotal(int(totalSize)).
		WithTitle("Uploading").
		WithShowPercentage(true).
		WithShowCount(true).
		Start()
	defer progressBar.Stop()

	stream := clients.Relay.Stream(ctx)

	// Send StreamInit first
	if err := stream.Send(&v1.StreamRequest{
		Payload: &v1.StreamRequest_Init{
			Init: &v1.StreamInit{
				ShareId:     shareID,
				ShareSecret: shareSecret,
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

			// Update progress bar
			progressBar.Add(n)

		} else if msg.GetComplete() != nil {
			// Server indicates completion
			pterm.Success.Println(msgUploadComplete)
			break
		}
	}

	// Close stream
	if err := stream.CloseRequest(); err != nil {
		return fmt.Errorf("failed to close stream: %w", err)
	}

	pterm.Success.Println(msgShareCreatedSuccess)
	pterm.Info.Printfln(msgShareIDInfo, shareID)
	pterm.Info.Printfln(msgShareSecretInfo, shareSecret)
	pterm.Info.Printfln(msgDownloadCommand, os.Args[0], shareID, shareSecret)

	return nil
}

// =============================================================================
// PULL COMMAND
// =============================================================================

func pullCommand(ctx context.Context, shareID uint64, shareSecret string) error {
	// Create HTTP client with device certificate
	httpClient, err := createHTTPClient(cfg.CACertPath, cfg.DeviceCert, cfg.DeviceKey)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	shareClient := v1connect.NewShareServiceClient(httpClient, cfg.ServerURL)
	relayClient := v1connect.NewRelayServiceClient(httpClient, cfg.ServerURL)

	pterm.Info.Printfln(msgGettingShareDetails, shareSecret)

	// Get share details to know what files to download (if we have a valid ID)
	var files []*v1.DetailResponse_File
	var totalExpectedSize int64
	if shareID > 0 {
		detailResp, err := shareClient.Detail(ctx, connect.NewRequest(&v1.DetailRequest{
			Id: shareID,
		}))
		if err != nil {
			pterm.Warning.Printfln(msgShareDetailsFailed, err)
		} else {
			files = detailResp.Msg.GetFiles()
			pterm.Info.Printfln(msgShareContainsFiles, len(files))
			// Calculate total size for progress bar
			for _, f := range files {
				totalExpectedSize += int64(f.GetSize())
			}
		}
	}

	pterm.Info.Printfln(msgStartingDownload, shareSecret)

	// Create progress bar if we know the total size
	var progressBar *pterm.ProgressbarPrinter
	if totalExpectedSize > 0 {
		progressBar, _ = pterm.DefaultProgressbar.
			WithTotal(int(totalExpectedSize)).
			WithTitle("Downloading").
			WithShowPercentage(true).
			WithShowCount(true).
			Start()
		defer progressBar.Stop()
	}

	// Start bidirectional consume stream
	stream := relayClient.Consume(ctx)

	// Send ConsumeInit with share secret
	if err := stream.Send(&v1.ConsumeRequest{
		Payload: &v1.ConsumeRequest_Init{
			Init: &v1.ConsumeInit{
				ShareId:     shareID,
				ShareSecret: shareSecret,
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
				pterm.Info.Printfln(msgFileDownloadStarted, path, filesReceived)
			}

			// Write chunk data at offset
			if len(data) > 0 {
				if _, err := file.WriteAt(data, int64(offset)); err != nil {
					return fmt.Errorf("failed to write to file %s: %w", filePath, err)
				}
			}

			totalBytesReceived += uint64(len(data))

			// Update progress bar if available
			if progressBar != nil && len(data) > 0 {
				progressBar.Add(len(data))
			}

		} else if msg.GetComplete() != nil {
			// Server indicates completion
			pterm.Success.Println(msgDownloadComplete)
			break
		}
	}

	// Close all files
	for path, file := range openFiles {
		if file != nil {
			file.Close()
			openFiles[path] = nil
			pterm.Success.Printfln(msgFileComplete, path)
		}
	}

	// Close stream
	if err := stream.CloseRequest(); err != nil {
		return fmt.Errorf("failed to close stream: %w", err)
	}

	pterm.Success.Println(msgDownloadSuccess)
	pterm.Info.Printfln(msgFilesReceived, filesReceived)
	pterm.Info.Printfln(msgTotalBytes, totalBytesReceived)

	return nil
}

// =============================================================================
// SUMMARY COMMAND
// =============================================================================

func summaryCommand(ctx context.Context, shareID uint64, shareSecret string) error {
	// Create clients with device certificate
	clients := createClientsOrDie(true)

	pterm.Info.Printfln(msgGettingStats, shareID, shareSecret)

	// Call Summary endpoint
	resp, err := clients.Relay.Summary(ctx, connect.NewRequest(&v1.SummaryRequest{
		ShareId:     shareID,
		ShareSecret: shareSecret,
	}))
	if err != nil {
		return fmt.Errorf("failed to get summary: %w", err)
	}

	files := resp.Msg.GetFiles()
	if len(files) == 0 {
		pterm.Info.Println(msgNoShareFiles)
		return nil
	}

	renderShareSummary(shareID, shareSecret, files)
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
