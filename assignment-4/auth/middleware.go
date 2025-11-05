package auth

import (
	"context"
	"crypto/x509"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/stupside/DATA-intensive/assignment-4/config"
	"github.com/stupside/DATA-intensive/assignment-4/crypto"
	"github.com/stupside/DATA-intensive/assignment-4/internal/errors"
	"github.com/stupside/DATA-intensive/assignment-4/internal/repository"
	"github.com/stupside/DATA-intensive/assignment-4/models"
)

var (
	certCtxKey   = &struct{}{}
	deviceCtxKey = &struct{}{}
)

// WithDevice adds a device to the context
func WithDevice(ctx context.Context, device *models.Device) context.Context {
	return context.WithValue(ctx, deviceCtxKey, device)
}

// DeviceFromContext extracts the device from context
func DeviceFromContext(ctx context.Context) (*models.Device, error) {
	device, ok := ctx.Value(deviceCtxKey).(*models.Device)
	if !ok {
		return nil, errors.Unauthenticated(ctx, "No device in context")
	}
	return device, nil
}

// WithCert adds a certificate to the context
func WithCert(ctx context.Context, cert *models.Certificate) context.Context {
	return context.WithValue(ctx, certCtxKey, cert)
}

// CertFromContext extracts the certificate from context
func CertFromContext(ctx context.Context) (*models.Certificate, error) {
	cert, ok := ctx.Value(certCtxKey).(*models.Certificate)
	if !ok {
		return nil, errors.Unauthenticated(ctx, "No certificate in context")
	}
	return cert, nil
}

// MTLSInterceptor verifies client certificates and adds them to context
func MTLSInterceptor(cfg *config.Config, certRepo repository.CertificateRepository, connectionRepo repository.ConnectionRepository) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract IP address
			ipAddress := extractIPAddress(r)

			// If no client certificate, allow request to proceed
			// (Onboarding endpoint will work, others can check CertFromContext)
			if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
				slog.InfoContext(r.Context(), "No client certificate provided", "ip", ipAddress)
				next.ServeHTTP(w, r)
				return
			}

			// Load and verify certificate using repository
			cert, err := verifyCertificate(r.Context(), cfg, certRepo, r.TLS.PeerCertificates[0])

			// Record connection attempt (async to not block request)
			if cert != nil {
				go recordConnection(context.Background(), connectionRepo, cert.DeviceID, ipAddress, err == nil)
			}

			if err != nil {
				slog.InfoContext(r.Context(), "Certificate verification failed", "error", err, "ip", ipAddress)
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			slog.InfoContext(r.Context(), "Found certificate in database", "cert_id", cert.ID, "device_id", cert.DeviceID, "ip", ipAddress)

			ctx := WithCert(r.Context(), cert)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AuthInterceptor extracts device from certificate context (works for both unary and streaming)
func AuthInterceptor(cfg *config.Config, deviceRepo repository.DeviceRepository) connect.Interceptor {
	return &authInterceptor{
		deviceRepo: deviceRepo,
	}
}

type authInterceptor struct {
	deviceRepo repository.DeviceRepository
}

var _ connect.Interceptor = (*authInterceptor)(nil)

func (i *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, ar connect.AnyRequest) (connect.AnyResponse, error) {
		ctx, err := i.authenticateFromCert(ctx)
		if err != nil {
			return nil, err
		}
		return next(ctx, ar)
	}
}

func (i *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next // Not used on server side
}

func (i *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		ctx, err := i.authenticateFromCert(ctx)
		if err != nil {
			return err
		}
		return next(ctx, conn)
	}
}

// authenticateFromCert is the shared authentication logic using repository
func (i *authInterceptor) authenticateFromCert(ctx context.Context) (context.Context, error) {
	cert, err := CertFromContext(ctx)
	if err != nil {
		return ctx, errors.Unauthenticated(ctx, "Authentication failed: no certificate in context")
	}

	slog.InfoContext(ctx, "Found certificate in context", "cert_id", cert.ID, "device_id", cert.DeviceID)

	device, err := i.deviceRepo.FindByID(ctx, cert.DeviceID)
	if err != nil {
		return ctx, errors.Unauthenticated(ctx, "Authentication failed: device not found")
	}

	slog.InfoContext(ctx, "Found device in database", "device_id", device.ID)

	return WithDevice(ctx, device), nil
}

// verifyCertificate verifies the client certificate and looks it up in the database using repository
func verifyCertificate(ctx context.Context, cfg *config.Config, certRepo repository.CertificateRepository, clientCert *x509.Certificate) (*models.Certificate, error) {
	// Load CA certificate
	caCert, err := crypto.LoadCACertificate(cfg.TLS.CA.CertPath)
	if err != nil {
		return nil, err
	}

	// Verify client certificate
	roots := x509.NewCertPool()
	roots.AddCert(caCert)

	if _, err := clientCert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}); err != nil {
		return nil, err
	}

	// Extract public key
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(clientCert.PublicKey)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Looking up certificate by public key", "pubkey_len", len(pubKeyBytes))

	// Look up certificate by public key using repository
	cert, err := certRepo.FindByPublicKey(ctx, pubKeyBytes)
	if err != nil {
		return nil, err
	}

	return cert, nil
}

// extractIPAddress extracts the client IP from the request
func extractIPAddress(r *http.Request) string {
	// Check X-Forwarded-For header (if behind proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take first IP in chain
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fallback to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// recordConnection creates a connection record (runs async)
func recordConnection(ctx context.Context, repo repository.ConnectionRepository, deviceID uint64, ipAddress string, success bool) {
	conn := &models.Connection{
		DeviceID:  deviceID,
		IPAddress: ipAddress,
		Success:   success,
	}

	if err := repo.Create(ctx, conn); err != nil {
		slog.ErrorContext(ctx, "Failed to record connection", "error", err, "device_id", deviceID)
	}
}
