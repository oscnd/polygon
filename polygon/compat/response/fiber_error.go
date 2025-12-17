package response

import (
	"errors"
	"strings"

	"github.com/bsthun/gut"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"go.scnd.dev/open/polygon/package/flow"
)

func HandleError(c *fiber.Ctx, err error) error {
	// * construct success
	success := false

	// * case of `*fiber.Error`
	var fiberError *fiber.Error
	if errors.As(err, &fiberError) {
		return c.Status(fiberError.Code).JSON(&ErrorResponse{
			Success: &success,
			Message: &fiberError.Message,
		})
	}

	// * case of `*flow.Error`
	var polygonError *flow.Error
	if errors.As(err, &polygonError) {
		if polygonError.Items[0].Error != nil {
			estr := polygonError.Items[0].Error.Error()
			return c.Status(fiber.StatusBadRequest).JSON(&ErrorResponse{
				Success: &success,
				Message: polygonError.Items[0].Message,
				Error:   &estr,
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(&ErrorResponse{
			Success: gut.Ptr(false),
			Message: polygonError.Items[0].Message,
			Error:   nil,
		})
	}

	// * case of `validator.ValidationErrors`
	var valErr validator.ValidationErrors
	if errors.As(err, &valErr) {
		var lists []string
		for _, err := range valErr {
			lists = append(lists, err.Field()+" ("+err.Tag()+")")
		}

		message := strings.Join(lists[:], ", ")

		return c.Status(fiber.StatusBadRequest).JSON(&ErrorResponse{
			Success: gut.Ptr(false),
			Message: gut.Ptr("validation failed on " + message),
			Error:   gut.Ptr(valErr.Error()),
		})
	}

	return c.Status(fiber.StatusInternalServerError).JSON(&ErrorResponse{
		Success: gut.Ptr(false),
		Message: gut.Ptr("unknown server error"),
		Error:   gut.Ptr(err.Error()),
	})
}
