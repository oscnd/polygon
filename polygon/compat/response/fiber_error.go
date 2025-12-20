package response

import (
	"errors"
	"strings"

	"github.com/bsthun/gut"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"go.scnd.dev/open/polygon/package/span"
)

func HandleError(c fiber.Ctx, err error) error {
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

	// * case of `*span.Error`
	var spanError *span.Error
	if errors.As(err, &spanError) {
		if spanError.Items[0].Error != nil {
			estr := spanError.Items[0].Error.Error()
			return c.Status(fiber.StatusBadRequest).JSON(&ErrorResponse{
				Success: &success,
				Message: spanError.Items[0].Message,
				Error:   &estr,
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(&ErrorResponse{
			Success: gut.Ptr(false),
			Message: spanError.Items[0].Message,
			Error:   nil,
		})
	}

	// * case of `validator.ValidationErrors`
	var validatorErr validator.ValidationErrors
	if errors.As(err, &validatorErr) {
		var lists []string
		for _, err := range validatorErr {
			lists = append(lists, err.Field()+" ("+err.Tag()+")")
		}

		message := strings.Join(lists[:], ", ")

		return c.Status(fiber.StatusBadRequest).JSON(&ErrorResponse{
			Success: gut.Ptr(false),
			Message: gut.Ptr("validation failed on " + message),
			Error:   gut.Ptr(validatorErr.Error()),
		})
	}

	return c.Status(fiber.StatusInternalServerError).JSON(&ErrorResponse{
		Success: gut.Ptr(false),
		Message: gut.Ptr("unknown server error"),
		Error:   gut.Ptr(err.Error()),
	})
}
