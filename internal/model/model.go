package model

import "time"

const SchemaVersion = 2

type Scope struct {
	ID          string `json:"id"`
	Project     string `json:"project"`
	ProjectName string `json:"project_name"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	Resolver    string `json:"resolver"`
}

type Index struct {
	SchemaVersion int       `json:"schema_version"`
	GeneratedAt   time.Time `json:"generated_at"`
	Scope         Scope     `json:"scope"`
	Tools         []Tool    `json:"tools"`
}

type Example struct {
	Intent  string `json:"intent"`
	Command string `json:"command"`
}

type Tool struct {
	ID             string    `json:"id"`
	Family         string    `json:"family,omitempty"`
	Command        string    `json:"command"`
	ResolvedPath   string    `json:"resolved_path,omitempty"`
	Status         string    `json:"status"`
	SemanticSource string    `json:"semantic_source"`
	ResolverSource string    `json:"resolver_source"`
	Description    string    `json:"description,omitempty"`
	Capabilities   []string  `json:"capabilities,omitempty"`
	Intents        []string  `json:"intents,omitempty"`
	Examples       []Example `json:"examples,omitempty"`
	Risk           string    `json:"risk,omitempty"`
	ProjectDefined bool      `json:"project_defined,omitempty"`
	Managed        bool      `json:"managed,omitempty"`
	Version        string    `json:"version,omitempty"`
	VerifiedAt     time.Time `json:"verified_at,omitempty"`
}

// Candidate is deliberately smaller than Tool. Default agent-facing results
// omit full paths, environment metadata, catalog internals, and pseudo-confidence.
type Candidate struct {
	ID              string   `json:"id"`
	Command         string   `json:"command"`
	Claim           string   `json:"claim,omitempty"`
	Signal          Evidence `json:"signal"`
	DeclaredExample string   `json:"declared_example,omitempty"`
	Family          string   `json:"-"`
	Score           int      `json:"-"`
}

type Evidence struct {
	Semantics    string `json:"semantics"`
	Availability string `json:"availability"`
	Behavior     string `json:"behavior"`
	Match        string `json:"match"`
}

type FindResult struct {
	Scope  ResultScope `json:"scope"`
	Match  *Candidate  `json:"match"`
	Status string      `json:"status,omitempty"`
}

type ResultScope struct {
	ID      string `json:"id"`
	Project string `json:"project"`
}
