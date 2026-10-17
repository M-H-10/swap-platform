package domain

import (
	"errors"
	"testing"
)

func validProject() ProjectInput {
	return ProjectInput{
		Title:        "Online shop",
		Description:  "A small online shop for handmade goods.",
		Category:     CategoryWeb,
		Budget:       150_000_00, // 150 000 rubles
		DurationDays: 45,
		Complexity:   6,
		TeamSize:     3,
		KPIs: []KPI{
			{Name: "Launch", Target: "site is live", Weight: 60},
			{Name: "Sales", Target: "10 orders in the first month", Weight: 40},
		},
	}
}

func TestProjectInputValidate(t *testing.T) {
	tests := []struct {
		name      string
		change    func(in *ProjectInput)
		wantField string // "" means the input must be valid
	}{
		{"valid input", func(in *ProjectInput) {}, ""},
		{"no KPIs is fine", func(in *ProjectInput) { in.KPIs = nil }, ""},
		{"title too short", func(in *ProjectInput) { in.Title = "ab" }, "title"},
		{"description too short", func(in *ProjectInput) { in.Description = "short" }, "description"},
		{"unknown category", func(in *ProjectInput) { in.Category = "games" }, "category"},
		{"budget below minimum", func(in *ProjectInput) { in.Budget = 99_99 }, "budget"},
		{"negative budget", func(in *ProjectInput) { in.Budget = -5 }, "budget"},
		{"zero duration", func(in *ProjectInput) { in.DurationDays = 0 }, "duration_days"},
		{"complexity 11", func(in *ProjectInput) { in.Complexity = 11 }, "complexity"},
		{"no team", func(in *ProjectInput) { in.TeamSize = 0 }, "team_size"},
		{"KPI weights not 100", func(in *ProjectInput) { in.KPIs[1].Weight = 30 }, "kpis"},
		{"KPI without name", func(in *ProjectInput) { in.KPIs[0].Name = "" }, "kpis"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := validProject()
			tc.change(&in)
			err := in.Validate()

			if tc.wantField == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			var ve *ValidationError
			if !errors.As(err, &ve) || ve.Field != tc.wantField {
				t.Fatalf("expected validation error on %q, got %v", tc.wantField, err)
			}
		})
	}
}

func TestProjectFilterDefaults(t *testing.T) {
	f := ProjectFilter{Query: "  shop "}
	f.Normalize()
	if f.Status != ProjectOpen || f.Limit != 20 || f.Query != "shop" {
		t.Fatalf("unexpected defaults: %+v", f)
	}
	if err := f.Validate(); err != nil {
		t.Fatalf("defaults should be valid: %v", err)
	}

	f.CreatorID = "not-a-uuid"
	var ve *ValidationError
	if err := f.Validate(); !errors.As(err, &ve) || ve.Field != "creator_id" {
		t.Fatalf("expected creator_id error, got %v", err)
	}
}
