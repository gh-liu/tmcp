package complete

import "context"

type CandidateKind string

const (
	CandidateCommand CandidateKind = "command"
	CandidateFlag    CandidateKind = "flag"
	CandidateValue   CandidateKind = "value"
)

type Candidate struct {
	Value   string
	Display string
	Note    string
	Kind    CandidateKind
}

type Provider interface {
	Candidates(context.Context, string) ([]Candidate, error)
}
