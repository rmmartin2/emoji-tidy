package emojiseq

import "testing"

func TestFormat_CollapsesWhitespace(t *testing.T) {
	out, warnings, err := Format("  \U0001F600   \U0001F600  \n", Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	want := "\U0001F600 \U0001F600\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestFormat_ValidZWJFamily(t *testing.T) {
	family := "\U0001F468‍\U0001F469‍\U0001F467"
	out, _, err := Format(family, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != family+"\n" {
		t.Fatalf("got %q, want %q", out, family+"\n")
	}
}

func TestFormat_StrictRejectsDanglingZWJ(t *testing.T) {
	_, _, err := Format("\U0001F44D‍", Options{})
	if err == nil {
		t.Fatal("expected an error for a dangling zero-width joiner")
	}
}

func TestFormat_LenientRepairsDanglingZWJ(t *testing.T) {
	out, warnings, err := Format("\U0001F44D‍", Options{Lenient: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected exactly one warning, got %d: %v", len(warnings), warnings)
	}
	want := "\U0001F44D\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestFormat_ValidKeycapSequence(t *testing.T) {
	keycap := "1️⃣"
	out, _, err := Format(keycap, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != keycap+"\n" {
		t.Fatalf("got %q, want %q", out, keycap+"\n")
	}
}

func TestFormat_StrictRejectsOrphanVariationSelector(t *testing.T) {
	_, _, err := Format("️", Options{})
	if err == nil {
		t.Fatal("expected an error for an orphan variation selector")
	}
}

func TestFormat_ValidFlagPair(t *testing.T) {
	flag := "\U0001F1FA\U0001F1F8" // regional indicators U and S
	out, _, err := Format(flag, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != flag+"\n" {
		t.Fatalf("got %q, want %q", out, flag+"\n")
	}
}
