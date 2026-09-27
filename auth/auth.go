package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/arnaudovproject/pisigo"
	"github.com/golang-jwt/jwt/v5"
)

const (
	ClaimsKey = "auth_claims"
	UserKey   = "auth_user"
	RolesKey  = "auth_roles"
)

var (
	ErrInvalidToken = errors.New("auth: invalid token")
	ErrMissingToken = errors.New("auth: missing token")
)

type Claims struct {
	UserID string         `json:"uid"`
	Roles  []string       `json:"roles"`
	Extra  map[string]any `json:"extra,omitempty"`
	jwt.RegisteredClaims
}

type JWTConfig struct {
	Secret     []byte
	Issuer     string
	Audience   string
	TTL        time.Duration
	TokenLookup func(c *pisigo.Context) string
}

func DefaultJWTConfig(secret string) JWTConfig {
	return JWTConfig{
		Secret: []byte(secret),
		TTL:    time.Hour,
		TokenLookup: func(c *pisigo.Context) string {
			return c.BearerToken()
		},
	}
}

func Issue(cfg JWTConfig, userID string, roles []string, extra map[string]any) (string, error) {
	if len(cfg.Secret) == 0 {
		return "", errors.New("auth: empty secret")
	}
	ttl := cfg.TTL
	if ttl == 0 {
		ttl = time.Hour
	}
	now := time.Now()
	reg := jwt.RegisteredClaims{
		Issuer:    cfg.Issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}
	if cfg.Audience != "" {
		reg.Audience = jwt.ClaimStrings{cfg.Audience}
	}
	claims := Claims{
		UserID:           userID,
		Roles:            roles,
		Extra:            extra,
		RegisteredClaims: reg,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(cfg.Secret)
}

func Parse(cfg JWTConfig, tokenString string) (*Claims, error) {
	opts := []jwt.ParserOption{}
	if cfg.Issuer != "" {
		opts = append(opts, jwt.WithIssuer(cfg.Issuer))
	}
	if cfg.Audience != "" {
		opts = append(opts, jwt.WithAudience(cfg.Audience))
	}
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return cfg.Secret, nil
	}, opts...)
	if err != nil {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func JWT(cfg JWTConfig) pisigo.Middleware {
	if cfg.TokenLookup == nil {
		cfg.TokenLookup = func(c *pisigo.Context) string { return c.BearerToken() }
	}
	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			token := cfg.TokenLookup(c)
			if token == "" {
				return pisigo.ErrUnauthorized
			}
			claims, err := Parse(cfg, token)
			if err != nil {
				return pisigo.ErrUnauthorized
			}
			c.Set(ClaimsKey, claims)
			c.Set(UserKey, claims.UserID)
			c.Set(RolesKey, claims.Roles)
			return next(c)
		}
	}
}

func APIKey(header string, valid func(key string) bool) pisigo.Middleware {
	if header == "" {
		header = "X-API-Key"
	}
	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			key := c.HeaderGet(header)
			if key == "" || valid == nil || !valid(key) {
				return pisigo.ErrUnauthorized
			}
			c.Set("api_key", key)
			return next(c)
		}
	}
}

func RequireRoles(roles ...string) pisigo.Middleware {
	needed := map[string]struct{}{}
	for _, r := range roles {
		needed[strings.ToLower(r)] = struct{}{}
	}
	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			raw, ok := c.Get(RolesKey)
			if !ok {
				return pisigo.ErrForbidden
			}
			have, _ := raw.([]string)
			for _, r := range have {
				if _, ok := needed[strings.ToLower(r)]; ok {
					return next(c)
				}
			}
			return pisigo.ErrForbidden
		}
	}
}

func ClaimsFromContext(c *pisigo.Context) (*Claims, bool) {
	raw, ok := c.Get(ClaimsKey)
	if !ok {
		return nil, false
	}
	claims, ok := raw.(*Claims)
	return claims, ok
}
