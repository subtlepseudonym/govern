package multierror

import (
	"fmt"
	"strings"
)

type MultiError []error

func (m MultiError) Error() string {
	var builder strings.Builder
	builder.WriteString("multierror:\n")

	for _, err := range m {
		fmt.Fprintf(&builder, "  err: %w\n", err)
	}

	return builder.String()
}

func (m MultiError) ErrOrNil() error {
	if m != nil {
		return m
	}
	return nil
}
