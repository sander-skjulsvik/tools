package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// innermostTag matches a {@tag ...} with no tag nested inside it.
var innermostTag = regexp.MustCompile(`\{@(\w+)\s*([^{}]*)\}`)

// StripTags turns 5etools inline markup into plain text:
// {@spell fireball|xphb} → fireball, {@dc 15} → DC 15.
func StripTags(s string) string {
	for innermostTag.MatchString(s) {
		s = innermostTag.ReplaceAllStringFunc(s, func(m string) string {
			sub := innermostTag.FindStringSubmatch(m)
			tag, parts := sub[1], strings.Split(sub[2], "|")
			switch tag {
			case "dc":
				return "DC " + parts[0]
			case "hit":
				return "+" + strings.TrimPrefix(parts[0], "+")
			case "scaledamage", "scaledice":
				return parts[len(parts)-1]
			case "h":
				return "Hit: "
			}
			if len(parts) >= 3 && parts[2] != "" {
				return parts[2]
			}
			return parts[0]
		})
	}
	return s
}

func FormatTime(s *Spell) string {
	var parts []string
	for _, t := range s.Time {
		var p string
		switch t.Unit {
		case "action", "bonus", "reaction":
			p = t.Unit
			if t.Number != 1 {
				p = fmt.Sprintf("%d %s", t.Number, t.Unit)
			}
		case "minute":
			p = fmt.Sprintf("%d min", t.Number)
		case "hour":
			p = fmt.Sprintf("%d hr", t.Number)
		default:
			p = fmt.Sprintf("%d %s", t.Number, t.Unit)
		}
		parts = append(parts, p)
	}
	return strings.Join(parts, " or ")
}

func FormatDuration(s *Spell) string {
	var parts []string
	for _, d := range s.Duration {
		switch d.Type {
		case "instant":
			parts = append(parts, "Instant")
		case "permanent":
			parts = append(parts, "Until dispelled")
		case "special":
			parts = append(parts, "Special")
		case "timed":
			unit := map[string]string{"round": "rnd", "minute": "min", "hour": "hr", "day": "day", "turn": "turn", "week": "wk", "year": "yr"}[d.Duration.Type]
			if unit == "" {
				unit = d.Duration.Type
			}
			parts = append(parts, fmt.Sprintf("%d %s", d.Duration.Amount, unit))
		default:
			parts = append(parts, d.Type)
		}
	}
	return strings.Join(parts, " or ")
}

var areaShapes = map[string]string{
	"cone": "cone", "cube": "cube", "emanation": "emanation", "hemisphere": "hemisphere",
	"line": "line", "radius": "radius", "sphere": "sphere",
}

func FormatRange(s *Spell) string {
	r := s.Range
	dist := func() string {
		switch r.Distance.Type {
		case "self":
			return "Self"
		case "touch":
			return "Touch"
		case "sight":
			return "Sight"
		case "unlimited":
			return "Unlimited"
		case "feet":
			return fmt.Sprintf("%d ft", r.Distance.Amount)
		case "miles":
			if r.Distance.Amount == 1 {
				return "1 mile"
			}
			return fmt.Sprintf("%d miles", r.Distance.Amount)
		}
		return r.Distance.Type
	}
	switch {
	case r.Type == "point":
		return dist()
	case areaShapes[r.Type] != "":
		return fmt.Sprintf("Self (%d-ft %s)", r.Distance.Amount, r.Type)
	case r.Type == "special":
		return "Special"
	}
	return r.Type
}

// FormatComponents gives "V S M"; M$ marks a material with a gp cost.
func FormatComponents(s *Spell) string {
	var parts []string
	if s.Components.V {
		parts = append(parts, "V")
	}
	if s.Components.S {
		parts = append(parts, "S")
	}
	if s.HasMaterial() {
		if s.MaterialCost() > 0 {
			parts = append(parts, "M$")
		} else {
			parts = append(parts, "M")
		}
	}
	return strings.Join(parts, " ")
}

func MaterialText(s *Spell) string {
	if !s.HasMaterial() {
		return ""
	}
	var text string
	if json.Unmarshal(s.Components.M, &text) == nil {
		return text
	}
	var m Material
	if json.Unmarshal(s.Components.M, &m) == nil {
		return m.Text
	}
	return ""
}

func LevelLabel(l int) string {
	if l == 0 {
		return "C"
	}
	return fmt.Sprint(l)
}

// RenderEntries flattens 5etools entries (strings, named sections, lists,
// tables) into plain-text paragraphs.
func RenderEntries(raw []json.RawMessage) []string {
	var out []string
	for _, r := range raw {
		out = append(out, renderEntry(r)...)
	}
	return out
}

func renderEntry(r json.RawMessage) []string {
	var str string
	if json.Unmarshal(r, &str) == nil {
		return []string{StripTags(str)}
	}
	var e struct {
		Type     string              `json:"type"`
		Name     string              `json:"name"`
		Entries  []json.RawMessage   `json:"entries"`
		Items    []json.RawMessage   `json:"items"`
		Entry    json.RawMessage     `json:"entry"`
		ColLabel []string            `json:"colLabels"`
		Rows     [][]json.RawMessage `json:"rows"`
	}
	if json.Unmarshal(r, &e) != nil {
		return nil
	}
	var out []string
	switch e.Type {
	case "list":
		for _, it := range e.Items {
			for _, line := range renderEntry(it) {
				out = append(out, "  - "+line)
			}
		}
	case "table":
		if len(e.ColLabel) > 0 {
			out = append(out, "  "+StripTags(strings.Join(e.ColLabel, " | ")))
		}
		for _, row := range e.Rows {
			var cells []string
			for _, c := range row {
				cells = append(cells, strings.Join(renderEntry(c), " "))
			}
			out = append(out, "  "+strings.Join(cells, " | "))
		}
	default:
		body := RenderEntries(e.Entries)
		if len(e.Entry) > 0 {
			body = append(renderEntry(e.Entry), body...)
		}
		if e.Name != "" && len(body) > 0 {
			body[0] = StripTags(e.Name) + ". " + body[0]
		}
		out = append(out, body...)
	}
	// Cells like {"type":"cell","roll":{...}} render empty; that's fine.
	return out
}
