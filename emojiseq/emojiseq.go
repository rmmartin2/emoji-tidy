// Package emojiseq checks and repairs the structural rules that hold an
// emoji sequence together: zero-width joiners, variation selectors, skin
// tone modifiers, regional indicator pairs, and keycap sequences.
//
// It does not validate against the full Unicode emoji-data property
// tables, so it will happily accept codepoint combinations that aren't
// real emoji (a ZWJ between two letters, say). What it catches is
// structural damage: a joiner with nothing to join, a modifier stuck to
// the wrong kind of thing, an unpaired flag half.
package emojiseq

import (
	"fmt"
	"strings"
)

const (
	zwj            rune = 0x200D
	vs15           rune = 0xFE0E
	vs16           rune = 0xFE0F
	skinToneLow    rune = 0x1F3FB
	skinToneHigh   rune = 0x1F3FF
	regionalLow    rune = 0x1F1E6
	regionalHigh   rune = 0x1F1FF
	keycapCombiner rune = 0x20E3
)

type runeKind int

const (
	kindNone runeKind = iota
	kindBase
	kindModifier
	kindVariation
	kindZWJ
	kindRegionalFirst
	kindKeycapBase
	kindKeycapVariation
	kindFlag
)

// Options controls how Format handles structurally invalid sequences.
type Options struct {
	// Lenient repairs violations (dropping the offending character and
	// recording a Warning) instead of failing with a ValidationErrors.
	Lenient bool
}

// Warning describes a repair made in lenient mode.
type Warning struct {
	Field   string
	Message string
}

func (w Warning) String() string {
	return fmt.Sprintf("%s: %s", w.Field, w.Message)
}

// ValidationError describes a single structural violation found in strict mode.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors collects every violation found across an input. Format
// keeps checking the rest of the input after a field fails so a single run
// reports everything wrong, rather than stopping at the first problem.
type ValidationErrors struct {
	Errors []*ValidationError
}

func (e *ValidationErrors) Error() string {
	parts := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		parts[i] = err.Error()
	}
	return strings.Join(parts, "; ")
}

// Format normalizes whitespace (collapsing runs of it to single spaces,
// trimming the ends) and checks each whitespace-separated field's emoji
// structure. In strict mode (the default) any violation makes Format
// return a *ValidationErrors and no output. In lenient mode violations are
// repaired in place and reported as warnings instead.
func Format(input string, opts Options) (string, []Warning, error) {
	fields := strings.Fields(input)
	out := make([]string, 0, len(fields))
	var warnings []Warning
	var verrs ValidationErrors

	for _, field := range fields {
		cleaned, fieldWarnings, fieldErrs := formatField(field, opts.Lenient)
		warnings = append(warnings, fieldWarnings...)
		if len(fieldErrs) > 0 {
			verrs.Errors = append(verrs.Errors, fieldErrs...)
			continue
		}
		if cleaned != "" {
			out = append(out, cleaned)
		}
	}

	if len(verrs.Errors) > 0 {
		return "", warnings, &verrs
	}
	return strings.Join(out, " ") + "\n", warnings, nil
}

// formatField walks one whitespace-separated field rune by rune, tracking
// what kind of thing was last emitted so joiners, modifiers, and selectors
// can be checked against whatever they're attached to.
func formatField(field string, lenient bool) (string, []Warning, []*ValidationError) {
	runes := []rune(field)
	out := make([]rune, 0, len(runes))
	var warnings []Warning
	var errs []*ValidationError
	last := kindNone

	// reject records a violation. It returns true if the violation is
	// fatal (strict mode, so the caller should stop and discard the
	// field) and false if it was repaired (lenient mode, so the caller
	// should skip the offending rune and carry on).
	reject := func(msg string) bool {
		if lenient {
			warnings = append(warnings, Warning{Field: field, Message: msg})
			return false
		}
		errs = append(errs, &ValidationError{Field: field, Message: msg})
		return true
	}

	for i, r := range runes {
		switch {
		case r == zwj:
			if last == kindFlag {
				if reject(fmt.Sprintf("position %d: zero-width joiner cannot follow a completed flag sequence", i)) {
					return "", warnings, errs
				}
				continue
			}
			if last != kindBase && last != kindModifier && last != kindVariation {
				if reject(fmt.Sprintf("position %d: zero-width joiner has no preceding emoji to join", i)) {
					return "", warnings, errs
				}
				continue
			}
			out = append(out, r)
			last = kindZWJ

		case r == vs15 || r == vs16:
			if last == kindFlag {
				if reject(fmt.Sprintf("position %d: variation selector cannot follow a completed flag sequence", i)) {
					return "", warnings, errs
				}
				continue
			}
			if last != kindBase && last != kindKeycapBase {
				if reject(fmt.Sprintf("position %d: variation selector is not attached to a base character", i)) {
					return "", warnings, errs
				}
				continue
			}
			out = append(out, r)
			if last == kindKeycapBase {
				last = kindKeycapVariation
			} else {
				last = kindVariation
			}

		case r >= skinToneLow && r <= skinToneHigh:
			if last == kindFlag {
				if reject(fmt.Sprintf("position %d: skin tone modifier cannot follow a completed flag sequence", i)) {
					return "", warnings, errs
				}
				continue
			}
			if last != kindBase {
				if reject(fmt.Sprintf("position %d: skin tone modifier is not attached to a base emoji", i)) {
					return "", warnings, errs
				}
				continue
			}
			out = append(out, r)
			last = kindModifier

		case r >= regionalLow && r <= regionalHigh:
			out = append(out, r)
			if last == kindRegionalFirst {
				last = kindFlag
			} else {
				last = kindRegionalFirst
			}

		case r == keycapCombiner:
			if last != kindKeycapBase && last != kindKeycapVariation {
				if reject(fmt.Sprintf("position %d: keycap combiner has no digit, '#', or '*' to attach to", i)) {
					return "", warnings, errs
				}
				continue
			}
			out = append(out, r)
			last = kindBase

		default:
			out = append(out, r)
			if isKeycapBase(r) {
				last = kindKeycapBase
			} else {
				last = kindBase
			}
		}
	}

	if last == kindZWJ {
		if reject("sequence ends with a dangling zero-width joiner") {
			return "", warnings, errs
		}
		if len(out) > 0 {
			out = out[:len(out)-1]
		}
	}
	if last == kindRegionalFirst {
		// A lone regional indicator is valid Unicode on its own (it just
		// renders as a boxed letter rather than half a flag), so this is
		// never a structural violation - it's a heads-up in both strict
		// and lenient mode, not something reject() should be able to fail
		// the field over.
		warnings = append(warnings, Warning{Field: field, Message: "sequence ends with an unpaired regional indicator"})
	}

	if len(errs) > 0 {
		return "", warnings, errs
	}
	return string(out), warnings, nil
}

func isKeycapBase(r rune) bool {
	return (r >= '0' && r <= '9') || r == '#' || r == '*'
}
