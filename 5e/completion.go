package main

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

// fishCompletion is printed by `5e completion fish`. Values that depend on the
// data (classes, subclasses, books, spell names) come from `5e __complete`,
// which reads the edition and class already on the command line.
const fishCompletion = `# fish completion for 5e. Install with:
#   5e completion fish > ~/.config/fish/completions/5e.fish

function __5e_spells
    __fish_seen_subcommand_from spells
end

complete -c 5e -f
complete -c 5e -n __fish_use_subcommand -a spells -d 'Search spells'
complete -c 5e -n __fish_use_subcommand -a completion -d 'Print shell completion'
complete -c 5e -n '__fish_seen_subcommand_from completion' -a fish

complete -c 5e -n __5e_spells -o e -x -a '(5e __complete editions)' -d 'Edition (required)'
complete -c 5e -n __5e_spells -o c -x -a '(5e __complete classes (commandline -opc))' -d Class
complete -c 5e -n __5e_spells -o sub -x -a '(5e __complete subclasses (commandline -opc))' -d 'Subclass (needs -c)'
complete -c 5e -n __5e_spells -o l -x -a '(5e __complete levels)' -d 'Level or range, e.g. 1-3'
complete -c 5e -n __5e_spells -o t -x -a '(5e __complete times)' -d 'Casting time'
complete -c 5e -n __5e_spells -o school -x -a '(5e __complete schools)' -d School
complete -c 5e -n __5e_spells -o src -x -a '(5e __complete sources (commandline -opc))' -d 'Only this book'
complete -c 5e -n __5e_spells -o data -r -F -d '5etools data directory'
complete -c 5e -n __5e_spells -o conc -d 'Only concentration'
complete -c 5e -n __5e_spells -o no-conc -d 'Only without concentration'
complete -c 5e -n __5e_spells -o ritual -d 'Only rituals'
complete -c 5e -n __5e_spells -o no-m -d 'No material component'
complete -c 5e -n __5e_spells -o no-costly -d 'No material with a gp cost'
complete -c 5e -n __5e_spells -o full -d 'Full spell text'
complete -c 5e -n __5e_spells -a '(5e __complete names (commandline -opc))'
`

// runComplete prints completion candidates, one per line, as "value\tdescription".
// args[0] is the kind; the rest is the command line so far.
func runComplete(args []string, w io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("__complete needs a kind")
	}
	kind, line := args[0], args[1:]
	switch kind {
	case "editions":
		fmt.Fprint(w, "14\t2014 rules\n24\t2024 rules\n")
		return nil
	case "levels":
		fmt.Fprint(w, "0\tCantrips\n")
		for l := 1; l <= 9; l++ {
			fmt.Fprintf(w, "%d\tLevel %d\n", l, l)
		}
		return nil
	case "times":
		fmt.Fprint(w, "action\nbonus\tBonus action\nreaction\nminute\tOne minute or longer\nhour\tOne hour or longer\n")
		return nil
	case "schools":
		for _, name := range sortedValues(schools) {
			fmt.Fprintln(w, strings.ToLower(name))
		}
		return nil
	}

	edition, _ := ParseEdition(flagValue(line, "e"))
	dir := flagValue(line, "data")
	if dir == "" {
		dir = defaultDataDir()
	}
	d, err := Load(dir)
	if err != nil {
		return err
	}
	var out []string
	switch kind {
	case "classes":
		out = lower(d.Classes(edition))
	case "subclasses":
		out = lower(d.Subclasses(edition, flagValue(line, "c")))
	case "sources":
		out = d.Sources(edition)
	case "names":
		out = d.Names(edition)
	default:
		return fmt.Errorf("__complete: unknown kind %q", kind)
	}
	for _, v := range out {
		fmt.Fprintln(w, v)
	}
	return nil
}

// flagValue finds -name value, --name value or -name=value in tokens.
func flagValue(tokens []string, name string) string {
	for i, t := range tokens {
		t = strings.TrimPrefix(strings.TrimPrefix(t, "-"), "-")
		if t == name && i+1 < len(tokens) {
			return tokens[i+1]
		}
		if v, ok := strings.CutPrefix(t, name+"="); ok {
			return v
		}
	}
	return ""
}

func lower(in []string) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = strings.ToLower(v)
	}
	return out
}

func sortedValues(m map[string]string) []string {
	var out []string
	for _, v := range m {
		out = append(out, v)
	}
	slices.Sort(out)
	return out
}

func sortedKeys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// inEdition reports whether a list entry belongs to the edition; "" means either.
func (d *Data) inEdition(e ListEntry, ed Edition) bool {
	got := d.classEdition(e)
	return got != "" && (ed == "" || got == string(ed))
}

// Classes lists the classes with spells in the edition ("" = both).
func (d *Data) Classes(ed Edition) []string {
	set := map[string]bool{}
	for _, lists := range d.Lists {
		for _, e := range lists {
			if d.inEdition(e, ed) {
				set[e.ClassName] = true
			}
		}
	}
	return sortedKeys(set)
}

// Subclasses lists the subclass short names of a class that add spells.
func (d *Data) Subclasses(ed Edition, class string) []string {
	set := map[string]bool{}
	for _, lists := range d.Lists {
		for _, e := range lists {
			if e.Subclass != "" && d.inEdition(e, ed) && (class == "" || strings.EqualFold(e.ClassName, class)) {
				set[e.Subclass] = true
			}
		}
	}
	return sortedKeys(set)
}

// Sources lists the books spells in the edition come from.
func (d *Data) Sources(ed Edition) []string {
	set := map[string]bool{}
	for _, s := range d.Spells {
		if ed == "" || d.inEditionSpell(s, ed) {
			set[s.Source] = true
		}
	}
	return sortedKeys(set)
}

// Names lists spell names in the edition.
func (d *Data) Names(ed Edition) []string {
	set := map[string]bool{}
	for _, s := range d.Spells {
		if ed == "" || d.inEditionSpell(s, ed) {
			set[s.Name] = true
		}
	}
	return sortedKeys(set)
}

func (d *Data) inEditionSpell(s *Spell, ed Edition) bool {
	_, ok := d.onList(s, Query{Edition: ed})
	return ok
}
