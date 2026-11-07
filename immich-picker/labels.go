package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
)

// labelInput is one entry of a labels file. Pointers tell a missing field from
// an empty one.
type labelInput struct {
	AssetID       *string   `json:"asset_id"`
	Model         *string   `json:"model"`
	PromptVersion *string   `json:"prompt_version"`
	Keep          *bool     `json:"keep"`
	Category      *string   `json:"category"`
	Description   *string   `json:"description"`
	Issues        *[]string `json:"issues"`
	EditNeeded    *bool     `json:"edit_needed"`
	EditNotes     string    `json:"edit_notes"`
	Score         *float64  `json:"score"`
	Notes         string    `json:"notes"`
}

func decodeLabel(raw json.RawMessage, known map[string]*Record) (*labelInput, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var input labelInput
	if err := decoder.Decode(&input); err != nil {
		return nil, err
	}
	var missing []string
	for name, absent := range map[string]bool{
		"asset_id":       input.AssetID == nil,
		"model":          input.Model == nil,
		"prompt_version": input.PromptVersion == nil,
		"keep":           input.Keep == nil,
		"category":       input.Category == nil,
		"description":    input.Description == nil,
		"issues":         input.Issues == nil,
		"edit_needed":    input.EditNeeded == nil,
	} {
		if absent {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		slices.Sort(missing)
		return nil, fmt.Errorf("missing fields: %s", strings.Join(missing, ", "))
	}
	if _, ok := known[*input.AssetID]; !ok {
		return nil, fmt.Errorf("unknown asset_id %q", *input.AssetID)
	}
	if *input.EditNeeded && strings.TrimSpace(input.EditNotes) == "" {
		return nil, fmt.Errorf("edit_needed is set without edit_notes")
	}
	return &input, nil
}

// ImportLabels validates every label first, so a bad file changes nothing.
func ImportLabels(catalog *Catalog, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("expected a JSON list of labels: %w", err)
	}
	records, err := catalog.RunRecords()
	if err != nil {
		return err
	}
	known := make(map[string]*Record, len(records))
	for _, record := range records {
		known[record.AssetID] = record
	}

	inputs := make([]*labelInput, 0, len(entries))
	var problems []string
	for index, entry := range entries {
		input, err := decodeLabel(entry, known)
		if err != nil {
			problems = append(problems, fmt.Sprintf("#%d: %v", index, err))
			continue
		}
		inputs = append(inputs, input)
	}
	if len(problems) > 0 {
		return fmt.Errorf("nothing imported:\n%s", strings.Join(problems, "\n"))
	}

	for _, input := range inputs {
		record := known[*input.AssetID]
		record.LLM = &Label{
			Model:         *input.Model,
			PromptVersion: *input.PromptVersion,
			Keep:          *input.Keep,
			Category:      *input.Category,
			Description:   *input.Description,
			Issues:        *input.Issues,
			EditNeeded:    *input.EditNeeded,
			EditNotes:     input.EditNotes,
			Score:         input.Score,
			Notes:         input.Notes,
			LabelledAt:    nowUTC(),
		}
		if err := catalog.Save(record); err != nil {
			return err
		}
	}
	fmt.Printf("Imported %d labels\n", len(inputs))
	return nil
}

// Status prints the catalog's progress.
func Status(catalog *Catalog) error {
	run, err := catalog.LoadRun()
	if err != nil {
		return err
	}
	records, err := catalog.RunRecords()
	if err != nil {
		return err
	}
	analyzed, labelled := 0, 0
	for _, record := range records {
		if record.Local != nil {
			analyzed++
		}
		if record.LLM != nil {
			labelled++
		}
	}
	fmt.Printf("Catalog %s: %s since %s, collected %s\n", catalog.root, run.PersonName, run.Since, run.CollectedAt)
	fmt.Printf("  %d photos, %d analyzed, %d labelled\n", len(records), analyzed, labelled)
	return nil
}
