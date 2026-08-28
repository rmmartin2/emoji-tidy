package emojiseq

import (
	"errors"
	"testing"
)

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

func TestFormat_LoneRegionalIndicatorIsNotFatal(t *testing.T) {
	// A single regional indicator is valid Unicode on its own - it renders
	// as a boxed letter rather than half a flag - so it must pass even in
	// strict mode, with a warning rather than an error.
	lone := "\U0001F1FA" // regional indicator U, unpaired
	out, warnings, err := Format(lone, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != lone+"\n" {
		t.Fatalf("got %q, want %q", out, lone+"\n")
	}
	if len(warnings) != 1 {
		t.Fatalf("expected exactly one warning, got %d: %v", len(warnings), warnings)
	}
}

func TestFormat_LoneRegionalIndicatorLenientMatchesStrict(t *testing.T) {
	lone := "\U0001F1FA"
	out, warnings, err := Format(lone, Options{Lenient: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != lone+"\n" {
		t.Fatalf("got %q, want %q", out, lone+"\n")
	}
	if len(warnings) != 1 {
		t.Fatalf("expected exactly one warning, got %d: %v", len(warnings), warnings)
	}
}

func TestFormat_StrictRejectsDoubleModifier(t *testing.T) {
	_, _, err := Format("\U0001F44D\U0001F3FB\U0001F3FC", Options{})
	if err == nil {
		t.Fatal("expected an error for a second modifier stacked on the first")
	}
}

func TestFormat_LenientRepairsDoubleModifier(t *testing.T) {
	out, warnings, err := Format("\U0001F44D\U0001F3FB\U0001F3FC", Options{Lenient: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "\U0001F44D\U0001F3FB\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected exactly one warning, got %d: %v", len(warnings), warnings)
	}
}

func TestFormat_StrictRejectsKeycapWithNoBase(t *testing.T) {
	_, _, err := Format("⃣", Options{})
	if err == nil {
		t.Fatal("expected an error for a keycap combiner with nothing to attach to")
	}
}

func TestFormat_MultipleFieldsAggregateErrors(t *testing.T) {
	_, _, err := Format("\U0001F44D‍ ️", Options{})
	if err == nil {
		t.Fatal("expected an error")
	}
	var verrs *ValidationErrors
	if !errors.As(err, &verrs) {
		t.Fatalf("expected *ValidationErrors, got %T", err)
	}
	if len(verrs.Errors) != 2 {
		t.Fatalf("expected both fields to report a violation, got %d: %v", len(verrs.Errors), verrs.Errors)
	}
}
