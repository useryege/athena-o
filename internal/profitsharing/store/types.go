package store

import (
	"errors"
	"time"
)

const (
	RoundPhaseDraft      = "draft"
	RoundPhaseCollecting = "collecting"
	RoundPhaseVoting     = "voting"
	RoundPhaseClosed     = "closed"

	ProposalStatusDraft     = "draft"
	ProposalStatusSubmitted = "submitted"

	requiredParticipantCount = 5
)

var (
	ErrNotConfigured            = errors.New("profit sharing store is not configured")
	ErrRoundNotFound            = errors.New("profit sharing round not found")
	ErrProposalNotFound         = errors.New("profit sharing proposal not found")
	ErrBallotNotFound           = errors.New("profit sharing ballot not found")
	ErrCandidateNotFound        = errors.New("profit sharing ballot candidate not found")
	ErrNotParticipant           = errors.New("requester is not a participant in the profit sharing round")
	ErrRoundAlreadyExists       = errors.New("profit sharing round already exists")
	ErrRevisionConflict         = errors.New("profit sharing revision conflict")
	ErrInvalidPhase             = errors.New("profit sharing operation is not allowed in the current phase")
	ErrInvalidParticipants      = errors.New("profit sharing participants are invalid")
	ErrInvalidProposal          = errors.New("profit sharing proposal is invalid")
	ErrIncompleteProposal       = errors.New("profit sharing proposal is incomplete")
	ErrNotAllProposalsSubmitted = errors.New("not all profit sharing proposals are submitted")
	ErrNotAllVotesSubmitted     = errors.New("not all profit sharing votes are submitted")
	ErrSelfVote                 = errors.New("self voting is not allowed")
)

type ParticipantInput struct {
	AccountID              string
	Username               string
	DisplayName            string
	DisplayOrder           int32
	BaselineResponsibility string
}

type Participant struct {
	RoundID                int64
	AccountID              string
	Username               string
	DisplayName            string
	DisplayOrder           int32
	BaselineResponsibility string
}

type Round struct {
	ID                 int64
	Slug               string
	Title              string
	Phase              string
	Revision           int64
	ActiveBallotNumber *int32
	FinalProposalID    *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	OpenedAt           *time.Time
	PublishedAt        *time.Time
	ClosedAt           *time.Time
}

type ProposalItemInput struct {
	ParticipantAccountID string
	Responsibility       string
	BasisPoints          *int32
}

type ProposalItem struct {
	RoundID              int64
	ProposalID           string
	ParticipantAccountID string
	Responsibility       string
	BasisPoints          *int32
}

type Proposal struct {
	ID              string
	RoundID         int64
	AuthorAccountID string
	Status          string
	AnonymousLabel  *string
	Revision        int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
	SubmittedAt     *time.Time
	Items           []ProposalItem
}

type Ballot struct {
	RoundID     int64
	Number      int32
	Status      string
	CreatedAt   time.Time
	ClosedAt    *time.Time
	Candidates  []Proposal
	VoteCounts  map[string]int64
	VotedCount  int64
	CurrentVote *Vote
}

type Vote struct {
	RoundID                 int64
	BallotNumber            int32
	VoterAccountID          string
	ProposalID              string
	ProposalAuthorAccountID string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type RoundSnapshot struct {
	Round        Round
	Participants []Participant
	Proposals    []Proposal
	Ballot       *Ballot
}

type CloseBallotResult struct {
	Round         Round
	RunoffCreated bool
}
