package domain

import (
	"errors"
	"testing"
)

func TestRegisterInputValidate(t *testing.T) {
	valid := RegisterInput{Username: "ali_99", Email: "ali@example.com", Password: "secret123", Role: RoleDeveloper}

	tests := []struct {
		name      string
		change    func(in *RegisterInput)
		wantField string // "" means the input must be valid
	}{
		{"valid input", func(in *RegisterInput) {}, ""},
		{"username too short", func(in *RegisterInput) { in.Username = "al" }, "username"},
		{"username with space", func(in *RegisterInput) { in.Username = "ali baba" }, "username"},
		{"bad email", func(in *RegisterInput) { in.Email = "not-an-email" }, "email"},
		{"email with name part", func(in *RegisterInput) { in.Email = "Ali <ali@example.com>" }, "email"},
		{"short password", func(in *RegisterInput) { in.Password = "1234567" }, "password"},
		{"unknown role", func(in *RegisterInput) { in.Role = "boss" }, "role"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := valid
			tc.change(&in)
			err := in.Validate()

			if tc.wantField == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			var ve *ValidationError
			if !errors.As(err, &ve) || ve.Field != tc.wantField {
				t.Fatalf("expected validation error on %q, got %v", tc.wantField, err)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	in := RegisterInput{Username: "  ali  ", Email: " Ali@Example.COM "}
	in.Normalize()
	if in.Username != "ali" || in.Email != "ali@example.com" || in.Role != RoleDeveloper {
		t.Fatalf("unexpected result: %+v", in)
	}
}
