package errors

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
)

// LogAndWrapError logs an error with context and returns a connect error with the original error wrapped.
// This eliminates the common pattern of:
//
//	slog.ErrorContext(ctx, "message", "error", err)
//	return connect.NewError(code, fmt.Errorf("message: %w", err))
func LogAndWrapError(ctx context.Context, code connect.Code, message string, err error, attrs ...any) error {
	// Build attributes slice with error
	allAttrs := append([]any{"error", err}, attrs...)
	slog.ErrorContext(ctx, message, allAttrs...)

	return connect.NewError(code, fmt.Errorf("%s: %w", message, err))
}

// LogAndWrap logs a message and returns a connect error without wrapping an underlying error.
// Use this when there's no underlying error to wrap.
func LogAndWrap(ctx context.Context, code connect.Code, message string, attrs ...any) error {
	slog.ErrorContext(ctx, message, attrs...)
	return connect.NewError(code, fmt.Errorf("%s", message))
}

// NotFound is a helper for "resource not found" errors.
// Example: errors.NotFound(ctx, "share", shareID)
func NotFound(ctx context.Context, resource string, id any) error {
	message := fmt.Sprintf("%s not found", resource)
	return LogAndWrap(ctx, connect.CodeNotFound, message, "id", id)
}

// Internal is a helper for internal server errors with an underlying cause.
// Example: errors.Internal(ctx, "Failed to create device", err)
func Internal(ctx context.Context, message string, err error) error {
	return LogAndWrapError(ctx, connect.CodeInternal, message, err)
}

// Unauthenticated is a helper for authentication errors.
// Example: errors.Unauthenticated(ctx, "Certificate not found")
func Unauthenticated(ctx context.Context, message string) error {
	return LogAndWrap(ctx, connect.CodeUnauthenticated, message)
}

// PermissionDenied is a helper for authorization errors.
// Example: errors.PermissionDenied(ctx, "Not authorized for this share", "share_id", shareID)
func PermissionDenied(ctx context.Context, message string, attrs ...any) error {
	return LogAndWrap(ctx, connect.CodePermissionDenied, message, attrs...)
}

// InvalidArgument is a helper for validation errors.
// Example: errors.InvalidArgument(ctx, "Invalid CSR format", err)
func InvalidArgument(ctx context.Context, message string, err error) error {
	return LogAndWrapError(ctx, connect.CodeInvalidArgument, message, err)
}

// InvalidArgumentMsg is a helper for validation errors without an underlying error.
// Example: errors.InvalidArgumentMsg(ctx, "First message must be ConsumeInit")
func InvalidArgumentMsg(ctx context.Context, message string) error {
	return LogAndWrap(ctx, connect.CodeInvalidArgument, message)
}
