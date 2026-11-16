package main

import (
	"encoding/json"
	"testing"
)

func TestStripTags(t *testing.T) {
	cases := map[string]string{
		"cast {@spell fireball|xphb}":                        "cast fireball",
		"a {@dc 15} save":                                    "a DC 15 save",
		"{@damage 8d6} fire":                                 "8d6 fire",
		"increases by {@scaledamage 8d6|3-9|1d6}":            "increases by 1d6",
		"{@variantrule Sphere [Area of Effect]|XPHB|Sphere}": "Sphere",
		"{@b {@i nested}}":                                   "nested",
	}
	for in, want := range cases {
		if got := StripTags(in); got != want {
			t.Errorf("StripTags(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseLevels(t *testing.T) {
	for in, want := range map[string][2]int{"": {0, 9}, "3": {3, 3}, "1-3": {1, 3}, "0-0": {0, 0}} {
		lo, hi, err := ParseLevels(in)
		if err != nil || lo != want[0] || hi != want[1] {
			t.Errorf("ParseLevels(%q) = %d, %d, %v; want %v", in, lo, hi, err, want)
		}
	}
	for _, in := range []string{"3-1", "10", "x", "1-"} {
		if _, _, err := ParseLevels(in); err == nil {
			t.Errorf("ParseLevels(%q) gave no error", in)
		}
	}
}

func spell(name, src string, level int, unit string, conc bool) *Spell {
	return &Spell{Name: name, Source: src, Level: level, Time: []Time{{Number: 1, Unit: unit}},
		Duration: []Duration{{Type: "timed", Concentration: conc}}}
}

// testData has one spell printed in both editions, one 2024-only spell, a
// subclass spell and an expanded-list spell.
func testData() *Data {
	old := spell("Smite", "PHB", 1, "bonus", true)
	old.ReprintedAs = []json.RawMessage{json.RawMessage(`"Smite|XPHB"`)}
	return &Data{
		Spells: []*Spell{
			old,
			spell("Smite", "XPHB", 1, "bonus", false),
			spell("New Ward", "XPHB", 2, "action", false),
			spell("Mark", "PHB", 1, "bonus", true),
			spell("Rite", "XGE", 1, "hour", false),
		},
		Lists: map[string][]ListEntry{
			"phb|smite":     {{ClassName: "Paladin", ClassSource: "PHB"}},
			"xphb|smite":    {{ClassName: "Paladin", ClassSource: "XPHB"}},
			"xphb|new ward": {{ClassName: "Paladin", ClassSource: "XPHB"}},
			"phb|mark":      {{ClassName: "Paladin", ClassSource: "PHB", Via: "Oath of Vengeance", Subclass: "Vengeance"}},
			"xge|rite":      {{ClassName: "Paladin", ClassSource: "PHB", Via: "XGE"}, {ClassName: "Paladin", ClassSource: "XPHB", Via: "XGE"}},
		},
		ClassEdition: map[string]string{"Paladin|PHB": "classic", "Paladin|XPHB": "one"},
		Superseded:   map[string]bool{"phb|smite": true},
		Reprints:     map[string]bool{"xphb|smite": true},
	}
}

func names(t *testing.T, d *Data, q Query) []string {
	t.Helper()
	if q.MaxLevel == 0 {
		q.MaxLevel = 9
	}
	rs, err := d.Search(q)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, r := range rs {
		out = append(out, r.Spell.Name+"|"+r.Spell.Source+"|"+r.Via)
	}
	return out
}

func equal(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestSearchEditions(t *testing.T) {
	d := testData()
	equal(t, names(t, d, Query{Edition: Edition2014, Class: "paladin"}), []string{"Rite|XGE|XGE", "Smite|PHB|"})
	equal(t, names(t, d, Query{Edition: Edition2024, Class: "Paladin"}), []string{"Rite|XGE|XGE", "Smite|XPHB|", "New Ward|XPHB|"})
	equal(t, names(t, d, Query{Edition: Edition2014, Class: "paladin", Subclass: "vengeance"}),
		[]string{"Mark|PHB|Oath of Vengeance", "Rite|XGE|XGE", "Smite|PHB|"})
}

func TestSearchFilters(t *testing.T) {
	d := testData()
	yes, no := true, false
	equal(t, names(t, d, Query{Edition: Edition2014, Class: "paladin", Concentration: &yes}), []string{"Smite|PHB|"})
	equal(t, names(t, d, Query{Edition: Edition2024, Class: "paladin", Concentration: &no, Times: []string{"bonus"}}), []string{"Smite|XPHB|"})
	equal(t, names(t, d, Query{Edition: Edition2024, Class: "paladin", MinLevel: 2, MaxLevel: 2}), []string{"New Ward|XPHB|"})
}

func TestSearchUnknownClass(t *testing.T) {
	d := testData()
	if _, err := d.Search(Query{Edition: Edition2014, Class: "palading", MaxLevel: 9}); err == nil {
		t.Error("unknown class gave no error")
	}
	if _, err := d.Search(Query{Edition: Edition2024, Class: "paladin", Subclass: "vengeance", MaxLevel: 9}); err == nil {
		t.Error("subclass with no 2024 spells gave no error")
	}
}
