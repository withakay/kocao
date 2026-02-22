package controlplaneapi

import "context"

type principalKey struct{}

// Principal is the authenticated caller identity for the request.
type Principal struct {
	TokenID string
	Scopes  map[string]struct{}
}

func withPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

func principalFrom(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(*Principal)
	return p, ok && p != nil
}
