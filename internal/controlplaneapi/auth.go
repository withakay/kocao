package controlplaneapi

import (
	"context"
	"net/http"
	"strings"
)

type Authenticator struct {
	tokens *TokenStore
}

func newAuthenticator(tokens *TokenStore) *Authenticator {
	return &Authenticator{tokens: tokens}
}

func bearerToken(r *http.Request) string {
	authz := strings.TrimSpace(r.Header.Get("Authorization"))
	if authz == "" {
		return ""
	}
	parts := strings.SplitN(authz, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func parseScopes(scopes string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, s := range strings.FieldsFunc(scopes, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' || r == '\t' }) {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out[s] = struct{}{}
	}
	return out
}

func hasScope(scopes map[string]struct{}, want string) bool {
	if scopes == nil {
		return false
	}
	if _, ok := scopes["*"]; ok {
		return true
	}
	_, ok := scopes[want]
	return ok
}

func (a *Authenticator) Authenticate(next http.Handler, _ *AuditStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := bearerToken(r)
		rec, err := a.tokens.Lookup(r.Context(), tok)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "auth lookup failed")
			return
		}
		if rec == nil {
			next.ServeHTTP(w, r)
			return
		}
		p := &Principal{TokenID: rec.ID, Scopes: parseScopes(rec.Scopes)}
		ctx := withPrincipal(r.Context(), p)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requireScopes(required []string, audit *AuditStore, describe func(*http.Request) (string, string, string)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := principalFrom(r.Context())
			act, rType, rID := describe(r)
			actor := "anonymous"
			if ok {
				actor = p.TokenID
			}
			if !ok {
				audit.Append(r.Context(), actor, act, rType, rID, "denied", map[string]any{"reason": "missing_token"})
				writeError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			for _, req := range required {
				if !hasScope(p.Scopes, req) {
					audit.Append(r.Context(), actor, act, rType, rID, "denied", map[string]any{"reason": "missing_scope", "required": req})
					writeError(w, http.StatusForbidden, "insufficient scope")
					return
				}
			}
			audit.Append(r.Context(), actor, act, rType, rID, "allowed", nil)
			next.ServeHTTP(w, r)
		})
	}
}

func principal(ctx context.Context) string {
	if p, ok := principalFrom(ctx); ok {
		return p.TokenID
	}
	return "anonymous"
}
