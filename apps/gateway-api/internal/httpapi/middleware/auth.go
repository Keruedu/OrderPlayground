package middleware

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"order-playground/apps/gateway-api/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type TokenVerifier struct {
	issuerURL string
	jwksBase  string
	clientID  string
	jwksURL   string
	client    *http.Client

	mu           sync.RWMutex
	keys         map[string]*rsa.PublicKey
	lastRefresh  time.Time
	refreshEvery time.Duration
}

type discoveryDocument struct {
	JWKSURI string `json:"jwks_uri"`
}

type jwksDocument struct {
	Keys []jsonWebKey `json:"keys"`
}

type jsonWebKey struct {
	KID string `json:"kid"`
	KTY string `json:"kty"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewTokenVerifier(ctx context.Context, issuerURL, jwksBase, clientID string) (*TokenVerifier, error) {
	verifier := &TokenVerifier{
		issuerURL:    strings.TrimRight(issuerURL, "/"),
		jwksBase:     strings.TrimRight(jwksBase, "/"),
		clientID:     clientID,
		client:       &http.Client{Timeout: 5 * time.Second},
		keys:         map[string]*rsa.PublicKey{},
		refreshEvery: 10 * time.Minute,
	}
	if err := verifier.refresh(ctx); err != nil {
		return nil, err
	}
	return verifier, nil
}

func (v *TokenVerifier) Verify(ctx context.Context, raw string) (domain.UserClaims, error) {
	if time.Since(v.lastRefresh) > v.refreshEvery {
		_ = v.refresh(ctx)
	}

	token, err := jwt.Parse(raw, func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		v.mu.RLock()
		key := v.keys[kid]
		v.mu.RUnlock()
		if key == nil {
			if err := v.refresh(ctx); err != nil {
				return nil, err
			}
			v.mu.RLock()
			key = v.keys[kid]
			v.mu.RUnlock()
		}
		if key == nil {
			return nil, fmt.Errorf("signing key not found")
		}
		return key, nil
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(v.issuerURL))
	if err != nil {
		return domain.UserClaims{}, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return domain.UserClaims{}, errors.New("invalid token claims")
	}

	if !matchesAudience(claims, v.clientID) {
		return domain.UserClaims{}, errors.New("token audience mismatch")
	}

	roles := extractRoles(claims)
	expiresAt, _ := claims.GetExpirationTime()
	return domain.UserClaims{
		Subject:           claimString(claims, "sub"),
		Username:          claimString(claims, "preferred_username"),
		Roles:             roles,
		RawTokenExpiresAt: expiresAt.Time,
	}, nil
}

func (v *TokenVerifier) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksBase+"/.well-known/openid-configuration", nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var discovery discoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return err
	}
	if discovery.JWKSURI == "" {
		return errors.New("jwks uri missing in discovery document")
	}

	jwksReq, err := http.NewRequestWithContext(ctx, http.MethodGet, discovery.JWKSURI, nil)
	if err != nil {
		return err
	}
	jwksResp, err := v.client.Do(jwksReq)
	if err != nil {
		return err
	}
	defer jwksResp.Body.Close()

	var document jwksDocument
	if err := json.NewDecoder(jwksResp.Body).Decode(&document); err != nil {
		return err
	}

	keys := map[string]*rsa.PublicKey{}
	for _, key := range document.Keys {
		if key.KTY != "RSA" || key.KID == "" || key.N == "" || key.E == "" {
			continue
		}
		pub, err := buildRSAPublicKey(key.N, key.E)
		if err != nil {
			return err
		}
		keys[key.KID] = pub
	}

	v.mu.Lock()
	v.jwksURL = discovery.JWKSURI
	v.keys = keys
	v.lastRefresh = time.Now()
	v.mu.Unlock()
	return nil
}

func Auth(verifier *TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		user, err := verifier.Verify(c.Request.Context(), strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.Request = c.Request.WithContext(WithUser(c.Request.Context(), user))
		c.Next()
	}
}

func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := UserFromContext(c.Request.Context())
		if !ok || !user.HasAnyRole(roles...) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}

func buildRSAPublicKey(modulus, exponent string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(modulus)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(exponent)
	if err != nil {
		return nil, err
	}
	eInt := 0
	for _, b := range eBytes {
		eInt = eInt<<8 + int(b)
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: eInt,
	}, nil
}

func matchesAudience(claims jwt.MapClaims, clientID string) bool {
	if claimString(claims, "azp") == clientID {
		return true
	}
	aud, ok := claims["aud"]
	if !ok {
		return false
	}
	switch value := aud.(type) {
	case string:
		return value == clientID
	case []any:
		for _, item := range value {
			if str, ok := item.(string); ok && str == clientID {
				return true
			}
		}
	}
	return false
}

func extractRoles(claims jwt.MapClaims) []string {
	var roles []string
	realmAccess, ok := claims["realm_access"].(map[string]any)
	if !ok {
		return roles
	}
	rawRoles, ok := realmAccess["roles"].([]any)
	if !ok {
		return roles
	}
	for _, role := range rawRoles {
		if str, ok := role.(string); ok {
			roles = append(roles, str)
		}
	}
	return roles
}

func claimString(claims jwt.MapClaims, key string) string {
	value, _ := claims[key].(string)
	return value
}
