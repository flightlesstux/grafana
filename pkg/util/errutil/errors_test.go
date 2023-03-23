package errutil

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBase_Is(t *testing.T) {
	baseNotFound := NewBase(StatusNotFound, "test.notFound")
	baseInternal := NewBase(StatusInternal, "test.internal")

	tests := []struct {
		Base            Base
		Other           error
		Expect          bool
		ExpectUnwrapped bool
	}{
		{
			Base:   Base{},
			Other:  errors.New(""),
			Expect: false,
		},
		{
			Base:   Base{},
			Other:  Base{},
			Expect: true,
		},
		{
			Base:   Base{},
			Other:  Error{},
			Expect: true,
		},
		{
			Base:   baseNotFound,
			Other:  baseNotFound,
			Expect: true,
		},
		{
			Base:   baseNotFound,
			Other:  baseNotFound.Errorf("this is an error derived from baseNotFound, it is considered to be equal to baseNotFound"),
			Expect: true,
		},
		{
			Base:   baseNotFound,
			Other:  baseInternal,
			Expect: false,
		},
		{
			Base:            baseInternal,
			Other:           fmt.Errorf("wrapped, like a burrito: %w", baseInternal.Errorf("oh noes")),
			Expect:          false,
			ExpectUnwrapped: true,
		},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf(
			"Base '%s' == '%s' of type %s = %v (%v unwrapped)",
			tc.Base.Error(),
			tc.Other.Error(),
			reflect.TypeOf(tc.Other),
			tc.Expect,
			tc.Expect || tc.ExpectUnwrapped,
		), func(t *testing.T) {
			assert.Equal(t, tc.Expect, tc.Base.Is(tc.Other), "direct comparison")
			assert.Equal(t, tc.Expect, errors.Is(tc.Base, tc.Other), "comparison using errors.Is with other as target")
			assert.Equal(t, tc.Expect || tc.ExpectUnwrapped, errors.Is(tc.Other, tc.Base), "comparison using errors.Is with base as target, should unwrap other")
		})
	}
}

func TestFrom(t *testing.T) {
	tests := []struct {
		Err                    error
		ExpectedIsGrafanaError bool
		ExpectedError          Error
	}{
		{
			Err:                    NewBase(StatusNotFound, "test.notFound", WithPublicMessage("not found")),
			ExpectedIsGrafanaError: true,
			ExpectedError:          Error{LogMessage: "not found", Reason: StatusNotFound, MessageID: "test.notFound", PublicMessage: "not found", LogLevel: "debug"},
		},
		{
			Err:                    NewBase(StatusBadRequest, "test.notFound", WithPublicMessage("not found")).Errorf(""),
			ExpectedIsGrafanaError: true,
			ExpectedError:          Error{LogMessage: "", Reason: StatusBadRequest, MessageID: "test.notFound", PublicMessage: "not found", LogLevel: "debug"},
		},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("From(%s) = %s", tc.Err, tc.ExpectedError), func(t *testing.T) {
			gError, isGError := From(tc.Err)
			assert.Equal(t, tc.ExpectedIsGrafanaError, isGError)
			assert.Equal(t, &tc.ExpectedError, gError)
		})
	}
}
