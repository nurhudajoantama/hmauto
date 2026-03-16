package request

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// DecodeAndValidate decodes the JSON body of r into v, then validates it using
// struct field tags (e.g. `validate:"required,max=100"`).
//
// On JSON decode failure it returns an error prefixed with "invalid JSON:".
// On validation failure it returns a human-readable error listing each failing
// field and the violated constraint, e.g. "label: required; value: max".
func DecodeAndValidate(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if err := validate.Struct(v); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			msgs := make([]string, 0, len(ve))
			for _, e := range ve {
				msgs = append(msgs, fmt.Sprintf("%s: %s", strings.ToLower(e.Field()), e.Tag()))
			}
			return fmt.Errorf("%s", strings.Join(msgs, "; "))
		}
		return err
	}

	return nil
}
