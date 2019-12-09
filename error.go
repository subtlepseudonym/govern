package govern

import (
	"fmt"
	"strings"
)

// Change uses the error interface to allow programmatic identification
// of public API changes
type Change error
type (
	// major
	PackageNameMismatch  Change
	TypeMismatch         Change
	SignatureMismatch    Change
	DeclarationRemoved   Change
	FieldRemoved         Change
	InterfaceGeneralized Change

	// minor
	DeclaractionAdded    Change
	FieldAdded           Change
	InterfaceSpecialized Change
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
