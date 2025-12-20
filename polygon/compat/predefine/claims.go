package predefine

import (
	"encoding/json"
	"time"

	"github.com/bsthun/gut"
	"github.com/golang-jwt/jwt/v5"
)

type OidcClaims struct {
	Id        *string `json:"sub"`
	Username  *string `json:"preferred_username"`
	FirstName *string `json:"given_name"`
	Lastname  *string `json:"family_name"`
	Picture   *string `json:"picture"`
	Email     *string `json:"email"`
}

type LoginClaims struct {
	UserId    *uint64    `json:"userId"`
	ExpiredAt *time.Time `json:"exp"`
}

func (r *LoginClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	if r.ExpiredAt == nil {
		return nil, nil
	}
	return &jwt.NumericDate{
		Time: *r.ExpiredAt,
	}, nil
}

func (r *LoginClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	return nil, nil
}

func (r *LoginClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil
}

func (r *LoginClaims) GetIssuer() (string, error) {
	return "", nil
}

func (r *LoginClaims) GetSubject() (string, error) {
	return gut.IdEncode(*r.UserId), nil
}

func (r *LoginClaims) GetAudience() (jwt.ClaimStrings, error) {
	return nil, nil
}

func (r *LoginClaims) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"userId": gut.IdEncode(*r.UserId),
	})
}

func (r *LoginClaims) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	userId, err := gut.IdDecode(raw["userId"].(string))
	if err != nil {
		return err
	}
	r.UserId = &userId
	return nil
}
