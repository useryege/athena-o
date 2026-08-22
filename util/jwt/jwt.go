package jwt

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	jwtgo "github.com/golang-jwt/jwt/v5"
)

// MapClaims converts a jwt.Claims to a MapClaims
func MapClaims(claims jwtgo.Claims) (jwtgo.MapClaims, error) {
	if mapClaims, ok := claims.(*jwtgo.MapClaims); ok {
		return *mapClaims, nil
	}
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return nil, err
	}
	var mapClaims jwtgo.MapClaims
	err = json.Unmarshal(claimsBytes, &mapClaims)
	if err != nil {
		return nil, err
	}
	return mapClaims, nil
}

// StringField extracts a field from the claims as a string
func StringField(claims jwtgo.MapClaims, fieldName string) string {
	if fieldIf, ok := claims[fieldName]; ok {
		if field, ok := fieldIf.(string); ok {
			return field
		}
	}
	return ""
}

// Float64Field extracts a field from the claims as a float64
func Float64Field(claims jwtgo.MapClaims, fieldName string) float64 {
	if fieldIf, ok := claims[fieldName]; ok {
		if field, ok := fieldIf.(float64); ok {
			return field
		}
	}
	return 0
}

func numField(m jwtgo.MapClaims, key string) (int64, error) {
	field, ok := m[key]
	if !ok {
		return 0, fmt.Errorf("token does not have %s claim", key)
	}
	switch val := field.(type) {
	case float64:
		return int64(val), nil
	case json.Number:
		return val.Int64()
	case int64:
		return val, nil
	default:
		return 0, fmt.Errorf("%s '%v' is not a number", key, val)
	}
}

// IssuedAt returns the issued at as an int64
func IssuedAt(m jwtgo.MapClaims) (int64, error) {
	return numField(m, "iat")
}

// IssuedAtTime returns the issued at as a time.Time
func IssuedAtTime(m jwtgo.MapClaims) (time.Time, error) {
	iat, err := IssuedAt(m)
	return time.Unix(iat, 0), err
}

// ExpirationTime returns the expiration as a time.Time
func ExpirationTime(m jwtgo.MapClaims) (time.Time, error) {
	exp, err := numField(m, "exp")
	return time.Unix(exp, 0), err
}

func Claims(in any) jwtgo.Claims {
	claims, ok := in.(jwtgo.Claims)
	if ok {
		return claims
	}
	return nil
}

func IsValid(token string) bool {
	return len(strings.SplitN(token, ".", 3)) == 3
}

// GetUserIdentifier returns the local JWT subject.
func GetUserIdentifier(c jwtgo.MapClaims) string {
	if c == nil {
		return ""
	}

	return StringField(c, "sub")
}
