package validate

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

type signupReq struct {
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age"   validate:"gte=18,lte=120"`
	Role  string `json:"role"  validate:"oneof=admin user"`
}

func TestValidate_OK(t *testing.T) {
	r := signupReq{Email: "a@b.com", Age: 30, Role: "user"}
	if err := Validate(r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_ReturnsErrors(t *testing.T) {
	r := signupReq{Email: "not-an-email", Age: 5, Role: "ghost"}
	err := Validate(r)
	if err == nil {
		t.Fatal("expected error")
	}

	es, ok := AsErrors(err)
	if !ok {
		t.Fatalf("expected Errors, got %T", err)
	}
	if len(es) != 3 {
		t.Errorf("expected 3 field errors, got %d (%v)", len(es), es)
	}
	fields := map[string]bool{}
	for _, fe := range es {
		fields[fe.Field] = true
	}
	for _, want := range []string{"email", "age", "role"} {
		if !fields[want] {
			t.Errorf("missing error for %q field", want)
		}
	}
}

func TestValidate_JSONFieldNames(t *testing.T) {
	r := signupReq{}
	err := Validate(r)
	es, _ := AsErrors(err)
	for _, fe := range es {
		if strings.ToLower(fe.Field) != fe.Field {
			t.Errorf("expected lowercase json field name, got %q", fe.Field)
		}
	}
}

func TestValidate_MessageFormat(t *testing.T) {
	r := signupReq{Age: 1}
	err := Validate(r)
	es, _ := AsErrors(err)
	var found bool
	for _, fe := range es {
		if fe.Field == "age" && strings.Contains(fe.Message, ">=") {
			found = true
		}
	}
	if !found {
		t.Errorf("age error message should mention >=, got %#v", es)
	}
}

func TestValidate_PtrInput(t *testing.T) {
	r := &signupReq{Email: "a@b.com", Age: 25, Role: "admin"}
	if err := Validate(r); err != nil {
		t.Fatalf("ptr input should validate: %v", err)
	}
}

func TestValidate_NonStruct(t *testing.T) {
	err := Validate(42)
	// Non-struct should produce some error, but NOT our Errors type.
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := AsErrors(err); ok {
		t.Errorf("non-struct should not return Errors; got %v", err)
	}
}

func TestRegisterRule(t *testing.T) {
	v := New()
	err := v.RegisterRule("even", func(fl validator.FieldLevel) bool {
		return fl.Field().Int()%2 == 0
	})
	if err != nil {
		t.Fatal(err)
	}

	type r struct {
		N int `json:"n" validate:"even"`
	}
	if err := v.Validate(r{N: 4}); err != nil {
		t.Errorf("4 should be even: %v", err)
	}
	if err := v.Validate(r{N: 3}); err == nil {
		t.Error("3 should fail even validation")
	}
}

func TestErrorsImplementsError(t *testing.T) {
	var e error = Errors{{Field: "x", Message: "bad x"}}
	if !strings.Contains(e.Error(), "bad x") {
		t.Errorf("Error() = %q", e.Error())
	}
	// Sanity: Errors must round-trip through errors.As.
	var es Errors
	if !errors.As(e, &es) || !reflect.DeepEqual(es, e.(Errors)) {
		t.Errorf("errors.As round-trip failed")
	}
}
