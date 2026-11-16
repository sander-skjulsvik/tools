package main

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// Edition is the 5etools edition name: "classic" for 2014, "one" for 2024.
type Edition string

const (
	Edition2014 Edition = "classic"
	Edition2024 Edition = "one"
)

func ParseEdition(s string) (Edition, error) {
	switch s {
	case "14", "2014":
		return Edition2014, nil
	case "24", "2024":
		return Edition2024, nil
	case "":
		return "", fmt.Errorf("-e is required: 14 or 24")
	}
	return "", fmt.Errorf("edition %q: use 14 or 24", s)
}

// ParseLevels reads "3" or "1-3" into an inclusive range.
func ParseLevels(s string) (lo, hi int, err error) {
	if s == "" {
		return 0, 9, nil
	}
	a, b, isRange := strings.Cut(s, "-")
	if lo, err = strconv.Atoi(strings.TrimSpace(a)); err != nil {
		return 0, 0, fmt.Errorf("level %q: want N or N-M", s)
	}
	hi = lo
	if isRange {
		if hi, err = strconv.Atoi(strings.TrimSpace(b)); err != nil {
			return 0, 0, fmt.Errorf("level %q: want N or N-M", s)
		}
	}
	if lo < 0 || hi > 9 || lo > hi {
		return 0, 0, fmt.Errorf("level %q: levels go from 0 (cantrip) to 9", s)
	}
	return lo, hi, nil
}

var timeAliases = map[string]string{
	"action": "action", "a": "action",
	"bonus": "bonus", "ba": "bonus", "b": "bonus",
	"reaction": "reaction", "r": "reaction",
	"minute": "minute", "min": "minute", "minutes": "minute",
	"hour": "hour", "hr": "hour", "hours": "hour",
}

// ParseTimes reads a comma list like "bonus,reaction".
func ParseTimes(s string) ([]string, error) {
	if s == "" {
		return nil, nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		u, ok := timeAliases[strings.ToLower(strings.TrimSpace(p))]
		if !ok {
			return nil, fmt.Errorf("casting time %q: use action, bonus, reaction, minute or hour", p)
		}
		out = append(out, u)
	}
	return out, nil
}

var schools = map[string]string{
	"A": "Abjuration", "C": "Conjuration", "D": "Divination", "E": "Enchantment",
	"V": "Evocation", "I": "Illusion", "N": "Necromancy", "T": "Transmutation", "P": "Psionic",
}

// ParseSchool matches a school by prefix ("evo", "Necromancy").
func ParseSchool(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	for code, name := range schools {
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(s)) {
			return code, nil
		}
	}
	return "", fmt.Errorf("school %q not known", s)
}

// Query holds the filters. Zero values mean "don't filter".
type Query struct {
	Edition       Edition
	Class         string
	Subclass      string
	MinLevel      int
	MaxLevel      int
	Times         []string
	Concentration *bool
	Ritual        bool
	School        string // one-letter code
	Name          string
	Source        string
	NoMaterial    bool
	NoCostly      bool
}

// Result is a matching spell and how it got onto the list.
type Result struct {
	Spell *Spell
	Via   string
}

// Search returns the matching spells sorted by level and name.
func (d *Data) Search(q Query) ([]Result, error) {
	if q.Subclass != "" && q.Class == "" {
		return nil, fmt.Errorf("-sub needs -c")
	}
	if q.Class != "" {
		if err := d.checkClass(q); err != nil {
			return nil, err
		}
	}
	var out []Result
	for _, s := range d.Spells {
		via, ok := d.onList(s, q)
		if !ok || !matches(s, q) {
			continue
		}
		out = append(out, Result{Spell: s, Via: via})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i].Spell, out[j].Spell
		if a.Level != b.Level {
			return a.Level < b.Level
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.Source < b.Source
	})
	return out, nil
}

// checkClass fails when the class or subclass has no spells in the edition,
// listing what does exist, so a typo doesn't look like an empty result.
func (d *Data) checkClass(q Query) error {
	classes := d.Classes(q.Edition)
	if !slices.ContainsFunc(classes, func(c string) bool { return strings.EqualFold(c, q.Class) }) {
		return fmt.Errorf("no %s class %q with spells; known: %s", editionName(q.Edition), q.Class, strings.Join(classes, ", "))
	}
	if q.Subclass == "" {
		return nil
	}
	subs := d.Subclasses(q.Edition, q.Class)
	if !slices.ContainsFunc(subs, func(s string) bool { return strings.EqualFold(s, q.Subclass) }) {
		return fmt.Errorf("no %s %s subclass %q with spells; known: %s", editionName(q.Edition), q.Class, q.Subclass, strings.Join(subs, ", "))
	}
	return nil
}

func editionName(e Edition) string {
	if e == Edition2024 {
		return "2024"
	}
	return "2014"
}

// onList reports whether the spell is available in the query's edition (and
// class), and through which list. A class's base list wins over an expanded
// or subclass list.
func (d *Data) onList(s *Spell, q Query) (string, bool) {
	key := spellKey(s.Source, s.Name)
	// Only one printing per edition: 2024 takes the newest, 2014 the
	// original and nothing from the 2024 Player's Handbook.
	if q.Edition == Edition2024 && d.Superseded[key] {
		return "", false
	}
	if q.Edition == Edition2014 && (d.Reprints[key] || strings.EqualFold(s.Source, "XPHB")) {
		return "", false
	}
	lists := d.Lists[key]
	if q.Class == "" {
		for _, e := range lists {
			if d.classEdition(e) == string(q.Edition) {
				return "", true
			}
		}
		// On no class list at all (monster or item spells): keep it.
		return "", len(lists) == 0
	}
	via, ok := "", false
	for _, e := range lists {
		if d.classEdition(e) != string(q.Edition) || !strings.EqualFold(e.ClassName, q.Class) {
			continue
		}
		if e.Subclass != "" && !strings.EqualFold(e.Subclass, q.Subclass) {
			continue
		}
		if e.Via == "" {
			return "", true
		}
		if !ok || e.Subclass != "" {
			via, ok = e.Via, true
		}
	}
	return via, ok
}

func matches(s *Spell, q Query) bool {
	if s.Level < q.MinLevel || s.Level > q.MaxLevel {
		return false
	}
	if len(q.Times) > 0 && !slices.ContainsFunc(s.Time, func(t Time) bool { return slices.Contains(q.Times, t.Unit) }) {
		return false
	}
	if q.Concentration != nil && s.Concentration() != *q.Concentration {
		return false
	}
	if q.Ritual && !s.Meta.Ritual {
		return false
	}
	if q.School != "" && s.School != q.School {
		return false
	}
	if q.Name != "" && !strings.Contains(strings.ToLower(s.Name), strings.ToLower(q.Name)) {
		return false
	}
	if q.Source != "" && !strings.EqualFold(s.Source, q.Source) {
		return false
	}
	if q.NoMaterial && s.HasMaterial() {
		return false
	}
	if q.NoCostly && s.MaterialCost() > 0 {
		return false
	}
	return true
}
