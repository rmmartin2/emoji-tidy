# emoji-tidy

Emoji sequences copied out of chat logs, spreadsheets, and old databases are
often structurally damaged in ways that don't show up until something
downstream trips over them: a zero-width joiner with nothing on one side, a
skin tone modifier stuck to a letter instead of an emoji, a variation
selector that ended up on plain text. Most tools either pass this through
untouched or don't look at it at all. emoji-tidy looks.

## What it checks

It reads text, splits it into whitespace-separated fields, and validates the
rules that hold an emoji sequence together:

- a zero-width joiner (U+200D) must sit between two emoji, never at the
  start, end, or next to plain text
- a variation selector (U+FE0E / U+FE0F) must follow a base character it can
  actually modify
- a skin tone modifier (U+1F3FB-U+1F3FF) must follow a base character,
  never another modifier, a joiner, or a variation selector
- regional indicator pairs (flags) and keycap sequences (digit/#/* +
  optional U+FE0F + U+20E3) are checked for completeness
- a lone trailing regional indicator is valid Unicode on its own (it
  renders as a boxed letter, not half a flag), so it's flagged with a
  warning rather than treated as a structural error, in both modes
- a completed flag (a paired regional indicator) or a completed keycap
  (digit/#/* + optional VS16 + U+20E3) is not a modifier base: a skin tone
  modifier, variation selector, or joiner right after one is rejected the
  same way as one attached to plain text

By default (strict mode) any violation is a fatal error and nothing is
printed. Pass `--lenient` to repair what it can instead - dropping orphan
joiners, stray modifiers, and unattached variation selectors - and print one
warning per repair on stderr rather than failing outright.

This checks structure, not semantics: it does not validate against the full
Unicode emoji-data property tables, so a joiner between two ordinary letters
currently passes as long as the joiner itself is correctly placed. See the
roadmap in the project notes for where that's headed.

## Usage

    $ echo "family: 👨‍👩‍👧   flag:  🇺🇸" | emoji-tidy
    family: 👨‍👩‍👧 flag: 🇺🇸

A dangling joiner is rejected by default:

    $ echo "👍‍" | emoji-tidy
    error: 👍‍: position 1: sequence ends with a dangling zero-width joiner

--lenient repairs it and warns instead of failing:

    $ echo "👍‍" | emoji-tidy --lenient
    👍
    (stderr: warning: 👍‍: sequence ends with a dangling zero-width joiner)

Reading from a file works the same way:

    $ emoji-tidy --lenient messy-input.txt > clean-output.txt

## Status

Early. The rule set above is everything it checks right now - no full
emoji-data conformance, no grapheme clustering for output, no JSON mode. It
works on what it covers.

## License

MIT, see LICENSE.
