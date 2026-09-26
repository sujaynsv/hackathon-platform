package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type EventStatus string

const (
	StatusDraft            EventStatus = "draft"
	StatusRegistrationOpen EventStatus = "registration_open"
	StatusSubmissionsOpen  EventStatus = "submissions_open"
	StatusJudging          EventStatus = "judging"
	StatusVoting           EventStatus = "voting"
	StatusResultsPublished EventStatus = "results_published"
	StatusArchived         EventStatus = "archived"
)

// allowedTransitions is the single source of truth for I15.
var allowedTransitions = map[EventStatus][]EventStatus{
	StatusDraft:            {StatusRegistrationOpen},
	StatusRegistrationOpen: {StatusSubmissionsOpen},
	StatusSubmissionsOpen:  {StatusJudging},
	StatusJudging:          {StatusVoting, StatusResultsPublished},
	StatusVoting:           {StatusResultsPublished},
	StatusResultsPublished: {StatusArchived},
	StatusArchived:         {},
}

var ErrInvalidTransition = errors.New("invalid state transition")
var ErrSlugTaken = errors.New("event slug already taken")

type Event struct {
	ID                   uuid.UUID
	Slug                 string
	Title                string
	Description          *string
	BannerURL            *string
	OrganizerID          uuid.UUID
	Status               EventStatus
	NormalizationStatus  string
	RegistrationOpensAt  *time.Time
	RegistrationClosesAt *time.Time
	SubmissionDeadlineAt *time.Time
	JudgingDeadlineAt    *time.Time
	VotingOpensAt        *time.Time
	VotingClosesAt       *time.Time
	MaxTeamSize          int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// Transition validates and applies a state transition (I15).
func (e *Event) Transition(target EventStatus) error {
	allowed := allowedTransitions[e.Status]
	for _, a := range allowed {
		if a == target {
			e.Status = target
			e.UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, e.Status, target)
}

func NewEvent(slug, title string, organizerID uuid.UUID, maxTeamSize int) (*Event, error) {
	if slug == "" {
		return nil, errors.New("slug is required")
	}
	if title == "" {
		return nil, errors.New("title is required")
	}
	if maxTeamSize < 1 {
		maxTeamSize = 5
	}
	now := time.Now().UTC()
	subDeadline := now.Add(30 * 24 * time.Hour)
	judgingDeadline := now.Add(45 * 24 * time.Hour)

	return &Event{
		ID:                   uuid.New(),
		Slug:                 slug,
		Title:                title,
		OrganizerID:          organizerID,
		Status:               StatusDraft,
		NormalizationStatus:  "awaiting_judging",
		MaxTeamSize:          maxTeamSize,
		SubmissionDeadlineAt: &subDeadline,
		JudgingDeadlineAt:    &judgingDeadline,
		CreatedAt:            now,
		UpdatedAt:            now,
	}, nil
}
