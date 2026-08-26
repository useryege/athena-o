package accountcredentials

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	jwtutil "github.com/useryege/athena/util/jwt"
)

const (
	// ClaimsIssuer is the fixed issuer for local Athena credentials.
	ClaimsIssuer = "athena"
	// TokenVersion is required on every current Athena session and API Key.
	TokenVersion              = 2
	minimumJWTSigningKeyBytes = 32
)

// ParsedToken is a verified local JWT plus its credential routing fields.
type ParsedToken struct {
	Claims          jwt.MapClaims
	Account         string
	Capability      Capability
	JTI             string
	IdentityBinding string
}

type localClaims struct {
	jwt.RegisteredClaims
	AthenaTokenVersion int    `json:"athenaTokenVersion"`
	IdentityBinding    string `json:"athenaIdentityBinding,omitempty"`
}

// JWTCodec is a stateless HMAC JWT signer and verifier.
type JWTCodec struct {
	signingKey []byte
}

// NewJWTCodec copies a minimum-strength HMAC signing key.
func NewJWTCodec(signingKey []byte) (*JWTCodec, error) {
	if len(signingKey) < minimumJWTSigningKeyBytes {
		return nil, fmt.Errorf("JWT signing key must contain at least %d bytes", minimumJWTSigningKeyBytes)
	}
	return &JWTCodec{signingKey: append([]byte(nil), signingKey...)}, nil
}

// Issue signs one v2 local login or API Key credential.
func (c *JWTCodec) Issue(account string, capability Capability, jti string, expiresIn int64, now time.Time, identityBinding string) (string, Token, error) {
	if account == "" || jti == "" {
		return "", Token{}, fmt.Errorf("token account and JTI are required")
	}
	if capability != CapabilityLogin && capability != CapabilityAPIKey {
		return "", Token{}, fmt.Errorf("unsupported token capability %q", capability)
	}
	if capability == CapabilityLogin && identityBinding == "" {
		return "", Token{}, fmt.Errorf("login token identity binding is required")
	}
	claims := localClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    ClaimsIssuer,
			NotBefore: jwt.NewNumericDate(now),
			Subject:   formatSubject(account, capability),
			ID:        jti,
		},
		AthenaTokenVersion: TokenVersion,
		IdentityBinding:    identityBinding,
	}
	metadata := Token{JTI: jti, IssuedAt: now.Unix()}
	if expiresIn > 0 {
		expiresAt := now.Add(time.Duration(expiresIn) * time.Second)
		claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(expiresAt)
		metadata.ExpiresAt = expiresAt.Unix()
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(c.signingKey)
	return signed, metadata, err
}

// Parse validates and projects an Athena-issued v2 login or API Key credential.
func (c *JWTCodec) Parse(tokenString string) (ParsedToken, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) { return c.signingKey, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(ClaimsIssuer),
		jwt.WithIssuedAt(),
	)
	if err != nil {
		return ParsedToken{}, err
	}
	if jwtutil.Float64Field(claims, "athenaTokenVersion") != TokenVersion {
		return ParsedToken{}, fmt.Errorf("unsupported Athena token version")
	}
	issuedAt, err := claims.GetIssuedAt()
	if err != nil {
		return ParsedToken{}, fmt.Errorf("parse token issued-at claim: %w", err)
	}
	if issuedAt == nil {
		return ParsedToken{}, fmt.Errorf("token issued-at claim is required")
	}
	notBefore, err := claims.GetNotBefore()
	if err != nil {
		return ParsedToken{}, fmt.Errorf("parse token not-before claim: %w", err)
	}
	if notBefore == nil {
		return ParsedToken{}, fmt.Errorf("token not-before claim is required")
	}
	rawSubject := jwtutil.GetUserIdentifier(claims)
	account, capability, err := parseSubject(rawSubject)
	if err != nil {
		return ParsedToken{}, err
	}
	jti := jwtutil.StringField(claims, "jti")
	if jti == "" {
		return ParsedToken{}, fmt.Errorf("token JTI is required")
	}
	identityBinding := jwtutil.StringField(claims, "athenaIdentityBinding")
	if capability == CapabilityLogin {
		if identityBinding == "" {
			return ParsedToken{}, fmt.Errorf("login token identity binding is required")
		}
		expiresAt, err := claims.GetExpirationTime()
		if err != nil {
			return ParsedToken{}, fmt.Errorf("parse login token expiration: %w", err)
		}
		if expiresAt == nil {
			return ParsedToken{}, fmt.Errorf("login token expiration is required")
		}
	}
	claims["sub"] = account
	return ParsedToken{
		Claims:          claims,
		Account:         account,
		Capability:      capability,
		JTI:             jti,
		IdentityBinding: identityBinding,
	}, nil
}

func formatSubject(account string, capability Capability) string {
	return account + ":" + string(capability)
}

func parseSubject(subject string) (string, Capability, error) {
	account, rawCapability, ok := strings.Cut(subject, ":")
	if !ok || account == "" || strings.Contains(rawCapability, ":") {
		return "", "", fmt.Errorf("token subject must use <account>:<capability>")
	}
	capability := Capability(rawCapability)
	if capability != CapabilityLogin && capability != CapabilityAPIKey {
		return "", "", fmt.Errorf("unsupported token capability %q", rawCapability)
	}
	return account, capability, nil
}
