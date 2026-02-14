package api

import "testing"

func TestValidateEmailRegex(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"a@b.com", true},
		{"a+b@b.com.br", true},
		{"", false},
		{"   ", false},
		{"a@", false},
		{"@b.com", false},
		{"a@b", false},
		{"a b@c.com", false},
	}
	for _, c := range cases {
		err := ValidateEmailRegex(c.in)
		if (err == nil) != c.want {
			t.Fatalf("email=%q wantOk=%v gotErr=%v", c.in, c.want, err)
		}
	}
}

func TestValidateGuardianAddress(t *testing.T) {
	ok := "Rua X\nBairro Y\nCidade Z\nSC\nBrasil\n89200000"
	if err := ValidateGuardianAddress(ok); err != nil {
		t.Fatalf("expected ok, got %v", err)
	}

	tooShort := "Rua X\nBairro Y\nCidade Z\nSC\nBrasil"
	if err := ValidateGuardianAddress(tooShort); err == nil {
		t.Fatal("expected error for missing CEP line")
	}

	badCep := "Rua X\nBairro Y\nCidade Z\nSC\nBrasil\n123"
	if err := ValidateGuardianAddress(badCep); err == nil {
		t.Fatal("expected error for invalid CEP digits")
	}

	badCep2 := "Rua X\nBairro Y\nCidade Z\nSC\nBrasil\n12.345-67"
	if err := ValidateGuardianAddress(badCep2); err == nil {
		t.Fatal("expected error for invalid CEP digits")
	}
}

