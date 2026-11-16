package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Spell is the part of a 5etools spell entry this tool reads.
type Spell struct {
	Name       string `json:"name"`
	Source     string `json:"source"`
	Page       int    `json:"page"`
	Level      int    `json:"level"`
	School     string `json:"school"`
	Time       []Time `json:"time"`
	Range      Range  `json:"range"`
	Components struct {
		V bool            `json:"v"`
		S bool            `json:"s"`
		M json.RawMessage `json:"m"`
	} `json:"components"`
	Duration []Duration `json:"duration"`
	Meta     struct {
		Ritual bool `json:"ritual"`
	} `json:"meta"`
	ReprintedAs        []json.RawMessage `json:"reprintedAs"`
	Entries            []json.RawMessage `json:"entries"`
	EntriesHigherLevel []json.RawMessage `json:"entriesHigherLevel"`
	Copy               json.RawMessage   `json:"_copy"`
}

type Time struct {
	Number    int    `json:"number"`
	Unit      string `json:"unit"`
	Condition string `json:"condition"`
	Note      string `json:"note"`
}

type Range struct {
	Type     string `json:"type"`
	Distance struct {
		Type   string `json:"type"`
		Amount int    `json:"amount"`
	} `json:"distance"`
}

type Duration struct {
	Type     string `json:"type"`
	Duration struct {
		Type   string `json:"type"`
		Amount int    `json:"amount"`
		UpTo   bool   `json:"upTo"`
	} `json:"duration"`
	Concentration bool     `json:"concentration"`
	Ends          []string `json:"ends"`
}

// Material holds a costed material component; plain-string materials leave Cost at 0.
type Material struct {
	Text    string `json:"text"`
	Cost    int    `json:"cost"`
	Consume any    `json:"consume"`
}

func (s *Spell) HasMaterial() bool {
	m := strings.TrimSpace(string(s.Components.M))
	return m != "" && m != "null" && m != "false"
}

// MaterialCost returns the cost in copper pieces, 0 when the material has no price.
func (s *Spell) MaterialCost() int {
	var m Material
	if json.Unmarshal(s.Components.M, &m) != nil {
		return 0
	}
	return m.Cost
}

func (s *Spell) Concentration() bool {
	for _, d := range s.Duration {
		if d.Concentration {
			return true
		}
	}
	return false
}

// ListEntry says how a spell got onto a class list.
type ListEntry struct {
	ClassName   string
	ClassSource string
	Via         string // "" for the base list, the defining book for an expanded list, or the subclass name
	Subclass    string // short name, set only for subclass lists
}

// lookupFile mirrors data/generated/gendata-spell-source-lookup.json:
// spell source → spell name (both lower case) → lists.
type lookupFile map[string]map[string]struct {
	// Either true (on the class's own list) or {definedInSources}
	// (added to the class list by another book).
	Class        map[string]map[string]json.RawMessage `json:"class"`
	ClassVariant map[string]map[string]definedIn       `json:"classVariant"`
	// classSource → class → subclassSource → shortName → {name}
	Subclass map[string]map[string]map[string]map[string]struct {
		Name string `json:"name"`
	} `json:"subclass"`
}

type definedIn struct {
	DefinedInSources []string `json:"definedInSources"`
}

// Data is everything loaded from a 5etools data directory.
type Data struct {
	Spells []*Spell
	// Lists maps "source|name" (lower case) to the class lists the spell is on.
	Lists map[string][]ListEntry
	// Superseded holds "source|name" (lower case) of spells that have a
	// newer printing; Reprints holds the newer printings themselves.
	Superseded map[string]bool
	Reprints   map[string]bool
	// ClassEdition maps "Name|SOURCE" to "classic" (2014) or "one" (2024).
	ClassEdition map[string]string
}

func spellKey(source, name string) string {
	return strings.ToLower(source) + "|" + strings.ToLower(name)
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// Load reads spells, class lists and class editions from a 5etools data directory.
func Load(dir string) (*Data, error) {
	if _, err := os.Stat(filepath.Join(dir, "spells", "index.json")); err != nil {
		return nil, fmt.Errorf("no 5etools data at %s (set -data or FIVETOOLS_DATA): %w", dir, err)
	}
	d := &Data{Lists: map[string][]ListEntry{}, ClassEdition: map[string]string{}, Superseded: map[string]bool{}, Reprints: map[string]bool{}}

	var index map[string]string
	if err := readJSON(filepath.Join(dir, "spells", "index.json"), &index); err != nil {
		return nil, err
	}
	for _, file := range index {
		var f struct {
			Spell []*Spell `json:"spell"`
		}
		if err := readJSON(filepath.Join(dir, "spells", file), &f); err != nil {
			return nil, err
		}
		for _, s := range f.Spell {
			if len(s.Copy) > 0 {
				return nil, fmt.Errorf("%s (%s) is a _copy entry, which this tool does not resolve", s.Name, s.Source)
			}
		}
		d.Spells = append(d.Spells, f.Spell...)
	}
	for _, s := range d.Spells {
		for _, raw := range s.ReprintedAs {
			uid, err := reprintUID(raw)
			if err != nil {
				return nil, fmt.Errorf("%s (%s) reprintedAs: %w", s.Name, s.Source, err)
			}
			name, src, ok := strings.Cut(uid, "|")
			if !ok {
				return nil, fmt.Errorf("%s (%s) reprintedAs %q has no source", s.Name, s.Source, uid)
			}
			d.Superseded[spellKey(s.Source, s.Name)] = true
			d.Reprints[spellKey(src, name)] = true
		}
	}

	var lookup lookupFile
	if err := readJSON(filepath.Join(dir, "generated", "gendata-spell-source-lookup.json"), &lookup); err != nil {
		return nil, err
	}
	for src, byName := range lookup {
		for name, l := range byName {
			key := src + "|" + name
			for classSrc, classes := range l.Class {
				for class, raw := range classes {
					e := ListEntry{ClassName: class, ClassSource: classSrc}
					if string(raw) != "true" {
						var v definedIn
						if err := json.Unmarshal(raw, &v); err != nil {
							return nil, fmt.Errorf("spell lookup %s/%s, class %s: %w", src, name, class, err)
						}
						e.Via = strings.Join(v.DefinedInSources, ",")
					}
					d.Lists[key] = append(d.Lists[key], e)
				}
			}
			for classSrc, classes := range l.ClassVariant {
				for class, v := range classes {
					d.Lists[key] = append(d.Lists[key], ListEntry{ClassName: class, ClassSource: classSrc, Via: strings.Join(v.DefinedInSources, ",")})
				}
			}
			for classSrc, classes := range l.Subclass {
				for class, bySubSrc := range classes {
					for _, subs := range bySubSrc {
						for short, sub := range subs {
							d.Lists[key] = append(d.Lists[key], ListEntry{ClassName: class, ClassSource: classSrc, Via: sub.Name, Subclass: short})
						}
					}
				}
			}
		}
	}

	classFiles, err := filepath.Glob(filepath.Join(dir, "class", "class-*.json"))
	if err != nil {
		return nil, err
	}
	for _, path := range classFiles {
		var f struct {
			Class []struct {
				Name    string `json:"name"`
				Source  string `json:"source"`
				Edition string `json:"edition"`
			} `json:"class"`
		}
		if err := readJSON(path, &f); err != nil {
			return nil, err
		}
		for _, c := range f.Class {
			d.ClassEdition[c.Name+"|"+strings.ToUpper(c.Source)] = c.Edition
		}
	}
	return d, nil
}

// reprintUID reads a reprintedAs item: "Name|SRC" or {"uid": "Name|SRC"}.
func reprintUID(raw json.RawMessage) (string, error) {
	var uid string
	if json.Unmarshal(raw, &uid) == nil {
		return uid, nil
	}
	var o struct {
		UID string `json:"uid"`
	}
	if err := json.Unmarshal(raw, &o); err != nil {
		return "", err
	}
	return o.UID, nil
}

// classEdition returns "classic", "one" or "" for a class on a list.
func (d *Data) classEdition(e ListEntry) string {
	return d.ClassEdition[e.ClassName+"|"+strings.ToUpper(e.ClassSource)]
}
