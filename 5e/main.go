// Command 5e searches D&D 5e data in a local 5etools checkout.
//
//	5e spells -e 24 -c paladin -l 1-3 -t bonus -conc
//	5e spells -e 14 fireball -full
//	5e completion fish > ~/.config/fish/completions/5e.fish
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

const usage = `Search D&D 5e data in a local 5etools checkout.

Usage:
  5e spells -e 14|24 [flags] [name]   search spells (5e spells -h for flags)
  5e completion fish                  print the fish completion script
`

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "5e:", err)
		os.Exit(1)
	}
}

func run(args []string, w io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(w, usage)
		return fmt.Errorf("no command given")
	}
	switch args[0] {
	case "spells":
		return runSpells(args[1:], w)
	case "completion":
		if len(args) != 2 || args[1] != "fish" {
			return fmt.Errorf("usage: 5e completion fish")
		}
		_, err := io.WriteString(w, fishCompletion)
		return err
	case "__complete":
		return runComplete(args[1:], w)
	case "-h", "-help", "--help", "help":
		fmt.Fprint(w, usage)
		return nil
	}
	return fmt.Errorf("unknown command %q\n\n%s", args[0], usage)
}

func defaultDataDir() string {
	if d := os.Getenv("FIVETOOLS_DATA"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "github.com", "5etools-mirror-3", "5etools-src", "data")
}

func runSpells(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("5e spells", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), `Search D&D 5e spells in a local 5etools data checkout.

Usage: 5e spells -e 14|24 [flags] [name]

Examples:
  5e spells -e 24 -c paladin -l 1-3            paladin spells, levels 1 to 3
  5e spells -e 14 -c paladin -sub vengeance    include Oath of Vengeance spells
  5e spells -e 24 -c cleric -t bonus -no-conc  bonus actions without concentration
  5e spells -e 14 -ritual -school div          divination rituals
  5e spells -e 24 fireball -full               full text

Columns: Lvl (C = cantrip), Time, Dur, C = concentration, R = ritual,
Comp (M$ = material with a gp cost), Src = book, Via = how it gets on the
class list when not the base list (expanded list book, or subclass).

Flags:
`)
		fs.PrintDefaults()
	}
	var (
		edition  = fs.String("e", "", "edition: 14 or 24 (required)")
		class    = fs.String("c", "", "class, e.g. paladin")
		sub      = fs.String("sub", "", "subclass short name, e.g. vengeance; adds its spells (needs -c)")
		level    = fs.String("l", "", "spell level or range, e.g. 2 or 1-3 (0 = cantrip)")
		times    = fs.String("t", "", "casting time, comma list: action, bonus, reaction, minute, hour")
		conc     = fs.Bool("conc", false, "only concentration spells")
		noConc   = fs.Bool("no-conc", false, "only spells without concentration")
		ritual   = fs.Bool("ritual", false, "only rituals")
		school   = fs.String("school", "", "school, prefix is enough: abj, conj, div, ench, evo, ill, necro, trans")
		source   = fs.String("src", "", "only spells from this book, e.g. XGE")
		noMat    = fs.Bool("no-m", false, "no material component")
		noCostly = fs.Bool("no-costly", false, "no material component with a gp cost")
		full     = fs.Bool("full", false, "print the full spell text")
		dataDir  = fs.String("data", defaultDataDir(), "5etools data directory (or set FIVETOOLS_DATA)")
	)
	// Allow the name before, between or after flags.
	var words []string
	for {
		if err := fs.Parse(args); err != nil {
			return err
		}
		if fs.NArg() == 0 {
			break
		}
		words = append(words, fs.Arg(0))
		args = fs.Args()[1:]
	}

	q := Query{Class: *class, Subclass: *sub, Ritual: *ritual, Source: *source, NoMaterial: *noMat, NoCostly: *noCostly, Name: strings.Join(words, " ")}
	var err error
	if q.Edition, err = ParseEdition(*edition); err != nil {
		return err
	}
	if q.MinLevel, q.MaxLevel, err = ParseLevels(*level); err != nil {
		return err
	}
	if q.Times, err = ParseTimes(*times); err != nil {
		return err
	}
	if q.School, err = ParseSchool(*school); err != nil {
		return err
	}
	if *conc && *noConc {
		return fmt.Errorf("-conc and -no-conc together match nothing")
	}
	if *conc || *noConc {
		v := *conc
		q.Concentration = &v
	}

	d, err := Load(*dataDir)
	if err != nil {
		return err
	}
	results, err := d.Search(q)
	if err != nil {
		return err
	}
	if *full {
		printFull(w, results)
	} else {
		printTable(w, results)
	}
	fmt.Fprintf(w, "\n%d spells (%s rules)\n", len(results), editionName(q.Edition))
	return nil
}

func mark(b bool, s string) string {
	if b {
		return s
	}
	return ""
}

func printTable(w io.Writer, results []Result) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "Lvl\tName\tTime\tDur\tC\tR\tRange\tComp\tSchool\tSrc\tVia")
	for _, r := range results {
		s := r.Spell
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			LevelLabel(s.Level), s.Name, FormatTime(s), FormatDuration(s),
			mark(s.Concentration(), "C"), mark(s.Meta.Ritual, "R"),
			FormatRange(s), FormatComponents(s), schools[s.School], s.Source, r.Via)
	}
	tw.Flush()
}

func printFull(w io.Writer, results []Result) {
	for i, r := range results {
		s := r.Spell
		if i > 0 {
			fmt.Fprintln(w, "\n"+strings.Repeat("─", 60))
		}
		level := "Cantrip"
		if s.Level > 0 {
			level = fmt.Sprintf("Level %d", s.Level)
		}
		fmt.Fprintf(w, "%s  (%s p.%d)\n", s.Name, s.Source, s.Page)
		fmt.Fprintf(w, "%s %s%s%s\n", level, schools[s.School], mark(s.Meta.Ritual, ", ritual"), mark(r.Via != "", ", via "+r.Via))
		fmt.Fprintf(w, "Time: %s", FormatTime(s))
		for _, t := range s.Time {
			if t.Condition != "" {
				fmt.Fprintf(w, " (%s)", StripTags(t.Condition))
			}
		}
		fmt.Fprintf(w, "\nRange: %s\nComponents: %s", FormatRange(s), FormatComponents(s))
		if m := MaterialText(s); m != "" {
			fmt.Fprintf(w, " (%s)", StripTags(m))
		}
		fmt.Fprintf(w, "\nDuration: %s%s\n\n", mark(s.Concentration(), "Concentration, "), FormatDuration(s))
		for _, p := range RenderEntries(s.Entries) {
			fmt.Fprintln(w, p)
		}
		for _, p := range RenderEntries(s.EntriesHigherLevel) {
			fmt.Fprintln(w, p)
		}
	}
}
