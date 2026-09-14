// Command emoji-tidy reads text and normalizes the emoji sequences in it,
// rejecting structurally broken ones unless --lenient is given.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/rmmartin2/emoji-tidy/emojiseq"
)

// jsonResult is the --json output shape. Warnings and Errors are always
// present as arrays (never null) so callers don't need to nil-check before
// ranging over them.
type jsonResult struct {
	Output   string                      `json:"output"`
	Warnings []emojiseq.Warning          `json:"warnings"`
	Errors   []*emojiseq.ValidationError `json:"errors"`
}

func main() {
	lenient := flag.Bool("lenient", false, "repair malformed sequences instead of rejecting them")
	jsonOutput := flag.Bool("json", false, "emit warnings, errors, and output as JSON on stdout instead of plain text")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: emoji-tidy [--lenient] [--json] [file]\n\n"+
			"Reads text from a file (or stdin if none is given), collapses whitespace,\n"+
			"and checks the structure of any emoji sequences it finds: joiners,\n"+
			"variation selectors, skin tone modifiers, flag pairs, keycaps. By default\n"+
			"a broken sequence is a fatal error and nothing is printed; --lenient\n"+
			"repairs what it can and warns on stderr instead. --json prints a single\n"+
			"JSON object on stdout (output, warnings, errors) rather than plain text\n"+
			"and stderr messages.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	var r io.Reader = os.Stdin
	if args := flag.Args(); len(args) > 0 {
		f, err := os.Open(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "emoji-tidy:", err)
			os.Exit(1)
		}
		defer f.Close()
		r = f
	}

	input, err := io.ReadAll(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, "emoji-tidy:", err)
		os.Exit(1)
	}

	out, warnings, err := emojiseq.Format(string(input), emojiseq.Options{Lenient: *lenient})

	if *jsonOutput {
		result := jsonResult{Output: out, Warnings: warnings, Errors: []*emojiseq.ValidationError{}}
		if warnings == nil {
			result.Warnings = []emojiseq.Warning{}
		}
		exitCode := 0
		if err != nil {
			exitCode = 1
			var verrs *emojiseq.ValidationErrors
			if errors.As(err, &verrs) {
				result.Errors = verrs.Errors
			} else {
				result.Errors = []*emojiseq.ValidationError{{Message: err.Error()}}
			}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if encErr := enc.Encode(result); encErr != nil {
			fmt.Fprintln(os.Stderr, "emoji-tidy:", encErr)
			os.Exit(1)
		}
		os.Exit(exitCode)
	}

	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	if err != nil {
		var verrs *emojiseq.ValidationErrors
		if errors.As(err, &verrs) {
			for _, e := range verrs.Errors {
				fmt.Fprintln(os.Stderr, "error:", e)
			}
		} else {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
		os.Exit(1)
	}

	fmt.Print(out)
}
