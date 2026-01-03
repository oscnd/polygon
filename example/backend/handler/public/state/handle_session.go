package stateEndpoint

import (
	"example/type/payload"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"go.scnd.dev/open/polygon/compat/predefine"
	"go.scnd.dev/open/polygon/compat/response"
)

// HandleSession
// @polygon handler
func (r *Handler) HandleSession(c fiber.Ctx) error {
	// * span
	s, _ := r.layer.With(c.Context())
	defer s.End()

	// * login claims
	var l *predefine.LoginClaims
	if jwtware.FromContext(c) != nil {
		l = jwtware.FromContext(c).Claims.(*predefine.LoginClaims)
	}

	_ = l

	// * parse body
	body := new(payload.SessionRequest)
	if err := c.Bind().Body(body); err != nil {
		return s.Error("failed to parse body", err)
	}
	s.Variable("body", body)

	// * response
	return c.JSON(response.Success(s, nil))
}
