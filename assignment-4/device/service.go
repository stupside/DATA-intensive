package device

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"connectrpc.com/connect"
	"github.com/stupside/DATA-intensive/assignment-4/config"
	"github.com/stupside/DATA-intensive/assignment-4/crypto"
	"github.com/stupside/DATA-intensive/assignment-4/models"
	v1 "github.com/stupside/DATA-intensive/assignment-4/modules/proto/gen/spec/v1"
	"github.com/stupside/DATA-intensive/assignment-4/modules/proto/gen/spec/v1/v1connect"
	"gorm.io/gorm"
)

// Service handles device onboarding business logic and implements the gRPC handler interface
type Service struct {
	db     *gorm.DB
	cfg    *config.Config
	caKey  ed25519.PrivateKey
	caCert *x509.Certificate
}

var _ v1connect.DeviceServiceHandler = (*Service)(nil)

// NewService creates a new device service with dependencies
func NewService(cfg *config.Config, db *gorm.DB) (*Service, error) {
	caCert, caKey, err := crypto.LoadCAWithKey(cfg.TLS.CA.CertPath, cfg.TLS.CA.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA: %w", err)
	}

	return &Service{
		db:     db,
		cfg:    cfg,
		caKey:  caKey,
		caCert: caCert,
	}, nil
}

// Onboard handles device onboarding requests
func (s *Service) Onboard(ctx context.Context, req *connect.Request[v1.OnboardRequest]) (*connect.Response[v1.OnboardResponse], error) {
	slog.InfoContext(ctx, "Processing onboarding request")

	// Parse and validate CSR
	csr, err := parseCSR(req.Msg.GetCsr())
	if err != nil {
		slog.ErrorContext(ctx, "CSR validation failed", "error", err)
		return nil, err
	}

	// Sign CSR to create certificate
	certPEM, serialNumber, err := signCSR(csr, s.caCert, s.caKey, s.cfg.Features.Onboarding.CertificateValidity)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to generate certificate", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate certificate: %w", err))
	}

	// Create user with certificate
	userID, err := s.createUserWithCertificate(ctx, csr.PublicKey)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Certificate issued successfully",
		"user_id", userID,
		"serial_number", serialNumber.String(),
		"subject", csr.Subject.String(),
	)

	return connect.NewResponse(&v1.OnboardResponse{Certificate: certPEM}), nil
}

// createUserWithCertificate creates a user and stores their certificate
func (s *Service) createUserWithCertificate(ctx context.Context, publicKey any) (uint, error) {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to marshal public key", "error", err)
		return 0, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to marshal public key: %w", err))
	}

	user := &models.User{
		Certificate: &models.Certificate{
			PublicKey: pubKeyBytes,
		},
	}
	if err := gorm.G[models.User](s.db).Create(ctx, user); err != nil {
		slog.ErrorContext(ctx, "Failed to create user during onboarding", "error", err)
		return 0, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create user: %w", err))
	}

	return user.ID, nil
}

// Certificate signing helper functions

const (
	serialNumberBits = 128
)

// parseCSR parses a PEM-encoded Certificate Signing Request
func parseCSR(csrPEM []byte) (*x509.CertificateRequest, error) {
	csrBlock, _ := pem.Decode(csrPEM)
	if csrBlock == nil || csrBlock.Type != "CERTIFICATE REQUEST" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid CSR PEM block"))
	}

	csr, err := x509.ParseCertificateRequest(csrBlock.Bytes)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to parse CSR: %w", err))
	}

	if err := csr.CheckSignature(); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid CSR signature: %w", err))
	}

	return csr, nil
}

// signCSR signs a Certificate Signing Request with the CA
func signCSR(csr *x509.CertificateRequest, caCert *x509.Certificate, caKey ed25519.PrivateKey, validity time.Duration) ([]byte, *big.Int, error) {
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), serialNumberBits))
	if err != nil {
		return nil, nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate serial number: %w", err))
	}

	certTemplate := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               csr.Subject,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(validity),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, certTemplate, caCert, csr.PublicKey, caKey)
	if err != nil {
		return nil, nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create certificate: %w", err))
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return certPEM, serialNumber, nil
}
