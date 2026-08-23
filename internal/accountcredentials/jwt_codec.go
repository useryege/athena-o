package accountcredentials

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	jwtutil "github.com/useryege/athena/util/jwt"
)

// ClaimsIssuer is the fixed issuer for local Athena credentials.
const ClaimsIssuer = "athena"

// ParsedToken is a verified local JWT plus its credential routing fields.
type ParsedToken struct {
	Claims          jwt.MapClaims
	Account         string
	Capability      Capability
	ID              string
	CredentialEpoch string
}

type localClaims struct {
	jwt.RegisteredClaims
	CredentialEpoch string `json:"athenaCredentialEpoch"`
}

// JWTCodec is a stateless HMAC JWT signer and verifier.
type JWTCodec struct {
	signingKey []byte
}

// NewJWTCodec copies the non-empty key loaded into catalog.
func NewJWTCodec(catalog *Catalog) (*JWTCodec, error) {
	if catalog == nil || len(catalog.signingKey) == 0 {
		return nil, fmt.Errorf("JWT signing key is empty")
	}
	return &JWTCodec{signingKey: append([]byte(nil), catalog.signingKey...)}, nil
}

// Issue signs one local login or API key credential using the supplied timestamp.
func (c *JWTCodec) Issue(account string, capability Capability, id string, expiresIn int64, now time.Time, credentialEpoch string) (string, Token, error) {
	claims := localClaims{RegisteredClaims: jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		Issuer:    ClaimsIssuer,
		NotBefore: jwt.NewNumericDate(now),
		Subject:   formatSubject(account, capability),
		ID:        id,
	}, CredentialEpoch: credentialEpoch}
	metadata := Token{ID: id, IssuedAt: now.Unix()}
	if expiresIn > 0 {
		expiresAt := now.Add(time.Duration(expiresIn) * time.Second)
		claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(expiresAt)
		metadata.ExpiresAt = expiresAt.Unix()
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(c.signingKey)
	return signed, metadata, err
}

// Parse validates and projects an Athena-issued login or API key credential.
func (c *JWTCodec) Parse(tokenString string) (ParsedToken, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) { return c.signingKey, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(ClaimsIssuer),
	)
	if err != nil {
		return ParsedToken{}, err
	}
	rawSubject := jwtutil.GetUserIdentifier(claims)
	account, capability := parseSubject(rawSubject)
	if account == "" {
		return ParsedToken{}, fmt.Errorf("token subject account is empty")
	}
	claims["sub"] = account
	return ParsedToken{
		Claims:          claims,
		Account:         account,
		Capability:      capability,
		ID:              jwtutil.StringField(claims, "jti"),
		CredentialEpoch: jwtutil.StringField(claims, "athenaCredentialEpoch"),
	}, nil
}

func formatSubject(account string, capability Capability) string {
	return account + ":" + string(capability)
}

func parseSubject(subject string) (string, Capability) {
	capability := CapabilityAPIKey
	parts := strings.Split(subject, ":")
	if len(parts) > 1 {
		subject = parts[0]
		switch Capability(parts[1]) {
		case CapabilityLogin:
			capability = CapabilityLogin
		case CapabilityAPIKey:
			capability = CapabilityAPIKey
		}
	}
	return subject, capability
}
