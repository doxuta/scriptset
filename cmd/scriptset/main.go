// Command scriptset reports the UTS #39 restriction level and resolved
// script set of each argument.
//
// With no arguments it runs a built-in set of examples, so that
//
//	go run github.com/doxuta/scriptset/cmd/scriptset@latest
//
// prints something useful without the caller having to think of an input.
package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/doxuta/scriptset"
)

// demo pairs an example string with why it is here. Every line is a case that
// a mixed-script check built on the Script property gets wrong, or a level
// boundary worth seeing.
var demo = []struct{ s, why string }{
	{"paypal", "plain ASCII"},
	{"pаypal", "the 'a' is U+0430 CYRILLIC SMALL LETTER A"},
	{"日本語", "Han only: one script, four values in the resolved set"},
	{"한글漢字", "Hangul and Han: single-script via Kore"},
	{"ラーメンramen", "Latin + Japanese is an approved combination"},
	{"김치kimchi", "Latin + Korean is an approved combination"},
	{"abcабв", "Latin + Cyrillic is not"},
	{"abcไทย", "Latin + one other Recommended script"},
	{"а·б", "U+00B7 MIDDLE DOT carries 16 scripts, none of them Cyrillic"},
	{"hello world", "the space is not in the identifier profile"},
}

func main() {
	verbose := flag.Bool("v", false, "print the augmented script set of every character")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: scriptset [-v] [string ...]\n\n")
		fmt.Fprintf(os.Stderr,
			"Reports the UTS #39 restriction level and resolved script set of each\n"+
				"argument. With no arguments, runs a built-in set of examples.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	fmt.Printf("scriptset — UTS #39 section 5, Unicode %s\n\n", scriptset.UnicodeVersion)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	// The string goes last: tabwriter counts bytes-as-cells, so a
	// double-width CJK column would throw off every column after it.
	fmt.Fprintln(w, "LEVEL\tRESOLVED SET\tSINGLE\tSTRING")

	args := flag.Args()
	if len(args) == 0 {
		for _, d := range demo {
			row(w, d.s, d.why)
		}
	} else {
		for _, s := range args {
			row(w, s, "")
		}
	}
	w.Flush()

	if *verbose {
		for _, s := range inputs(args) {
			fmt.Printf("\n%q\n", s)
			vw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, r := range s {
				fmt.Fprintf(vw, "  U+%04X\t%q\tallowed=%v\t%s\t\n",
					r, r, scriptset.Allowed(r), scriptset.Augmented(r))
			}
			vw.Flush()
		}
	}
}

func inputs(args []string) []string {
	if len(args) > 0 {
		return args
	}
	out := make([]string, 0, len(demo))
	for _, d := range demo {
		out = append(out, d.s)
	}
	return out
}

func row(w *tabwriter.Writer, s, why string) {
	single := "yes"
	if scriptset.IsMixedScript(s) {
		single = "no"
	}
	fmt.Fprintf(w, "%s\t%s\t%s\t%q", scriptset.LevelOf(s), scriptset.Resolve(s), single, s)
	if why != "" {
		fmt.Fprintf(w, "  — %s", why)
	}
	fmt.Fprintln(w)
}
