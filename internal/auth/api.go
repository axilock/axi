package auth

import "context"

type APIKeyCredentials struct {
	APIKey string
}

func (a *APIKeyCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + a.APIKey,
	}, nil
}

func (a *APIKeyCredentials) RequireTransportSecurity() bool {
	return false
}
