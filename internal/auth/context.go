package auth

import (
	"context"

	"google.golang.org/grpc/metadata"
)

type UserContext struct {
	MerchantID string
	UserID     string
	Role       string
}

// ExtractMerchantID helper - simpler version assuming middleware populates metadata or context
func GetMerchantID(ctx context.Context) string {
	// Check if added to context by interceptor
	if val, ok := ctx.Value("merchant_id").(string); ok {
		return val
	}

	// Fallback to metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if val := md.Get("x-merchant-id"); len(val) > 0 {
			return val[0]
		}
	}
	return ""
}
