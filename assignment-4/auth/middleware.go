package auth

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/stupside/DATA-intensive/assignment-4/config"
	"github.com/stupside/DATA-intensive/assignment-4/crypto"
	"github.com/stupside/DATA-intensive/assignment-4/models"
	"gorm.io/gorm"
)

var (
	userCtxKey = &struct{}{}
	certCtxKey = &struct{}{}
)

// WithUser adds a user to the context
func WithUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userCtxKey, user)
}

// UserFromContext extracts the user from context
func UserFromContext(ctx context.Context) (*models.User, error) {
	user, ok := ctx.Value(userCtxKey).(*models.User)
	if !ok {
		return nil, fmt.Errorf("no user in context")
	}
	return user, nil
}

// WithCert adds a certificate to the context
func WithCert(ctx context.Context, cert *models.Certificate) context.Context {
	return context.WithValue(ctx, certCtxKey, cert)
}

// CertFromContext extracts the certificate from context
func CertFromContext(ctx context.Context) (*models.Certificate, error) {
	cert, ok := ctx.Value(certCtxKey).(*models.Certificate)
	if !ok {
		return nil, fmt.Errorf("no certificate in context")
	}
	return cert, nil
}

// MTLSInterceptor verifies client certificates and adds them to context
func MTLSInterceptor(cfg *config.Config, db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If no client certificate, allow request to proceed
			// (Onboarding endpoint will work, others can check CertFromContext)
			if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
				slog.InfoContext(r.Context(), "No client certificate provided")
				next.ServeHTTP(w, r)
				return
			}

			// Load and verify certificate
			cert, err := verifyCertificate(r.Context(), cfg, db, r.TLS.PeerCertificates[0])
			if err != nil {
				slog.InfoContext(r.Context(), "Certificate verification failed", "error", err)
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			slog.InfoContext(r.Context(), "Found certificate in database", "cert_id", cert.ID, "user_id", cert.UserID)

			ctx := WithCert(r.Context(), cert)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AuthInterceptor extracts user from certificate context (works for both unary and streaming)
func AuthInterceptor(cfg *config.Config, db *gorm.DB) connect.Interceptor {
	return &authInterceptor{db: db}
}

type authInterceptor struct {
	db *gorm.DB
}

func (i *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, ar connect.AnyRequest) (connect.AnyResponse, error) {
		ctx, err := authenticateFromCert(ctx, i.db)
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
		ctx, err := authenticateFromCert(ctx, i.db)
		if err != nil {
			return err
		}
		return next(ctx, conn)
	}
}

// authenticateFromCert is the shared authentication logic
func authenticateFromCert(ctx context.Context, db *gorm.DB) (context.Context, error) {
	cert, err := CertFromContext(ctx)
	if err != nil {
		return ctx, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication failed: %w", err))
	}

	slog.InfoContext(ctx, "Found certificate in context", "cert_id", cert.ID, "user_id", cert.UserID)

	user, err := lookupUser(ctx, db, cert.UserID)
	if err != nil {
		slog.InfoContext(ctx, "User not found for certificate", "error", err, "user_id", cert.UserID)
		return ctx, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication failed: user not found"))
	}

	slog.InfoContext(ctx, "Found user in database", "user_id", user.ID)

	return WithUser(ctx, user), nil
}

// verifyCertificate verifies the client certificate and looks it up in the database
func verifyCertificate(ctx context.Context, cfg *config.Config, db *gorm.DB, clientCert *x509.Certificate) (*models.Certificate, error) {
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

	// Look up certificate by public key
	cert, err := gorm.G[models.Certificate](db).Where(&models.Certificate{PublicKey: pubKeyBytes}).First(ctx)
	if err != nil {
		return nil, err
	}

	return &cert, nil
}

// lookupUser finds a user by ID in the database
func lookupUser(ctx context.Context, db *gorm.DB, userID uint) (*models.User, error) {
	user, err := gorm.G[models.User](db).Where(&models.User{Model: gorm.Model{ID: userID}}).First(ctx)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
