package api

import "testing"

func TestBuildGuardianSignatureHTML_FontsAndEscaping(t *testing.T) {
	// escaping
	out := BuildGuardianSignatureHTML(`<b>Maria</b>`, "brush")
	if out == "" {
		t.Fatal("expected non-empty html")
	}
	if contains(out, "<b>") {
		t.Fatal("expected guardian name to be escaped")
	}
	// font selection
	out2 := BuildGuardianSignatureHTML("Maria", "dancing")
	if !contains(out2, "Dancing Script") {
		t.Fatalf("expected dancing script, got %q", out2)
	}
}

func TestFillContractBody_DataReplaceAndNoLocalReplace(t *testing.T) {
	body := "Local: Joinville\nData: [DATA]\nNome: [PACIENTE_NOME]\nAss: [ASSINATURA_RESPONSAVEL]"
	out := FillContractBody(body, nil, nil, "Clinica", "Objeto", "Tipo", "Mensal", "100", nil, nil, "", "", "", "", "Joinville", "11/02/2026")

	// [DATA] replaced
	if !contains(out, "11/02/2026") {
		t.Fatalf("expected date replaced, got %q", out)
	}
	// Local not replaced (template already contains it)
	if !contains(out, "Joinville") {
		t.Fatalf("expected local to remain, got %q", out)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (func() bool { return indexOf(s, sub) >= 0 })())
}

func indexOf(s, sub string) int {
	// tiny helper to avoid importing strings everywhere in tests
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
