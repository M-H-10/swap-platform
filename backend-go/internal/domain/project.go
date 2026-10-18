package domain

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Category is the kind of work a project is about.
type Category string

const (
	CategoryWeb       Category = "web"
	CategoryMobile    Category = "mobile"
	CategoryDesign    Category = "design"
	CategoryMarketing Category = "marketing"
	CategoryOther     Category = "other"
)

// ProjectStatus is where a project is in its life.
type ProjectStatus string

const (
	ProjectOpen       ProjectStatus = "open"        // waiting for a partner
	ProjectInProgress ProjectStatus = "in_progress" // a swap is active (stage 5)
	ProjectCompleted  ProjectStatus = "completed"
	ProjectFailed     ProjectStatus = "failed"
	ProjectCancelled  ProjectStatus = "cancelled"
)

// Money is always an integer number of kopecks: 1 ruble = 100 kopecks.
const (
	MinBudget = 100_00           // 100 rubles
	MaxBudget = 1_000_000_000_00 // 1 billion rubles
	MaxKPIs   = 10
)

// KPI is one measurable goal of a project. The weights of all KPIs add up to 100.
type KPI struct {
	Name   string `json:"name"`
	Target string `json:"target"`
	Weight int    `json:"weight"`
}

// Project is a piece of work a user wants to do with a partner.
type Project struct {
	ID                   string
	CreatorID            string
	CreatorUsername      string
	Title                string
	Description          string
	Category             Category
	Budget               int64 // kopecks
	DurationDays         int
	Complexity           int // 1 (easy) .. 10 (very hard)
	TeamSize             int
	KPIs                 []KPI
	SuccessProbabilityBP int // 500..9500, from the scoring formula
	Status               ProjectStatus
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// ProjectInput is what a user sends to create a project.
type ProjectInput struct {
	Title        string
	Description  string
	Category     Category
	Budget       int64
	DurationDays int
	Complexity   int
	TeamSize     int
	KPIs         []KPI
}

// ProjectUpdate holds the fields that may change after creation. A nil
// field means "leave it as it is". Budget, duration and the other numbers
// are fixed, because the success probability was calculated from them.
type ProjectUpdate struct {
	Title       *string
	Description *string
	KPIs        *[]KPI
}

// Normalize trims spaces so " Shop " and "Shop" are the same title.
func (in *ProjectInput) Normalize() {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.Category = Category(strings.ToLower(strings.TrimSpace(string(in.Category))))
	if in.KPIs == nil {
		in.KPIs = []KPI{}
	}
	for i := range in.KPIs {
		in.KPIs[i].Name = strings.TrimSpace(in.KPIs[i].Name)
		in.KPIs[i].Target = strings.TrimSpace(in.KPIs[i].Target)
	}
}

// Validate checks every field and returns the first problem it finds.
// The limits match the CHECK constraints of the projects table.
func (in ProjectInput) Validate() error {
	if n := utf8.RuneCountInString(in.Title); n < 3 || n > 120 {
		return &ValidationError{"title", "must be 3-120 characters"}
	}
	if n := utf8.RuneCountInString(in.Description); n < 10 || n > 5000 {
		return &ValidationError{"description", "must be 10-5000 characters"}
	}
	switch in.Category {
	case CategoryWeb, CategoryMobile, CategoryDesign, CategoryMarketing, CategoryOther:
	default:
		return &ValidationError{"category", "must be web, mobile, design, marketing or other"}
	}
	if in.Budget < MinBudget || in.Budget > MaxBudget {
		return &ValidationError{"budget", "must be between 10000 and 100000000000 kopecks (100 to 1 billion rubles)"}
	}
	if in.DurationDays < 1 || in.DurationDays > 3650 {
		return &ValidationError{"duration_days", "must be 1-3650"}
	}
	if in.Complexity < 1 || in.Complexity > 10 {
		return &ValidationError{"complexity", "must be 1-10"}
	}
	if in.TeamSize < 1 || in.TeamSize > 100 {
		return &ValidationError{"team_size", "must be 1-100"}
	}
	return validateKPIs(in.KPIs)
}

func validateKPIs(kpis []KPI) error {
	if len(kpis) > MaxKPIs {
		return &ValidationError{"kpis", "at most 10 KPIs"}
	}
	total := 0
	for _, k := range kpis {
		if n := utf8.RuneCountInString(k.Name); n < 1 || n > 100 {
			return &ValidationError{"kpis", "each name must be 1-100 characters"}
		}
		if n := utf8.RuneCountInString(k.Target); n < 1 || n > 200 {
			return &ValidationError{"kpis", "each target must be 1-200 characters"}
		}
		if k.Weight < 1 || k.Weight > 100 {
			return &ValidationError{"kpis", "each weight must be 1-100"}
		}
		total += k.Weight
	}
	if len(kpis) > 0 && total != 100 {
		return &ValidationError{"kpis", "the weights must add up to 100"}
	}
	return nil
}

// ProjectFilter is what the project list can be narrowed by.
type ProjectFilter struct {
	Status    ProjectStatus // empty = open projects only
	Category  Category      // empty = all categories
	Query     string        // words to look for in the title or description
	CreatorID string        // empty = everyone
	Limit     int
	Offset    int
}

// Normalize fills in the defaults.
func (f *ProjectFilter) Normalize() {
	f.Query = strings.TrimSpace(f.Query)
	if f.Status == "" {
		f.Status = ProjectOpen
	}
	if f.Limit == 0 {
		f.Limit = 20
	}
}

// Validate rejects values the database would not understand.
func (f ProjectFilter) Validate() error {
	switch f.Status {
	case ProjectOpen, ProjectInProgress, ProjectCompleted, ProjectFailed, ProjectCancelled:
	default:
		return &ValidationError{"status", "unknown status"}
	}
	switch f.Category {
	case "", CategoryWeb, CategoryMobile, CategoryDesign, CategoryMarketing, CategoryOther:
	default:
		return &ValidationError{"category", "unknown category"}
	}
	if utf8.RuneCountInString(f.Query) > 100 {
		return &ValidationError{"q", "must be at most 100 characters"}
	}
	if f.CreatorID != "" && !IsUUID(f.CreatorID) {
		return &ValidationError{"creator_id", "must be a user id"}
	}
	if f.Limit < 1 || f.Limit > 100 {
		return &ValidationError{"limit", "must be 1-100"}
	}
	if f.Offset < 0 {
		return &ValidationError{"offset", "must be 0 or more"}
	}
	return nil
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether s looks like a database id. Checking this first
// turns "/api/projects/abc" into a clean 404 instead of a database error.
func IsUUID(s string) bool {
	return uuidPattern.MatchString(s)
}
