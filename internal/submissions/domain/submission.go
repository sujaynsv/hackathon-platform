package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SubmissionStatus string

const (
	StatusDraft        SubmissionStatus = "draft"
	StatusSubmitted    SubmissionStatus = "submitted"
	StatusDisqualified SubmissionStatus = "disqualified"
)

var ErrAlreadySubmitted = errors.New("team already has a submission for this event")
var ErrSubmissionsNotOpen = errors.New("event is not accepting submissions")

type Submission struct {
	ID          uuid.UUID
	TeamID      uuid.UUID
	EventID     uuid.UUID
	TrackID     *uuid.UUID
	Title       string
	Description *string
	RepoURL     *string
	DemoURL     *string
	VideoURL    *string
	CoverURL    *string
	Status      SubmissionStatus
	FinalScore  *float64
	OverallRank *int
	TrackRank   *int
	SubmittedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewDraftSubmission(teamID, eventID uuid.UUID, trackID *uuid.UUID, title string) (*Submission, error) {
	if title == "" {
		return nil, errors.New("title is required")
	}
	now := time.Now().UTC()
	return &Submission{
		ID:        uuid.New(),
		TeamID:    teamID,
		EventID:   eventID,
		TrackID:   trackID,
		Title:     title,
		Status:    StatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// CheckCanCreate validates that the event is accepting submissions (I9).
func CheckCanCreate(eventStatus string) error {
	if eventStatus != "submissions_open" {
		return fmt.Errorf("%w: event status is '%s'", ErrSubmissionsNotOpen, eventStatus)
	}
	return nil
}

var ErrDeadlinePassed = errors.New("submission deadline has passed")
var ErrNotDraft = errors.New("only draft submissions can be edited")

// CheckCanEdit validates that the submission can still be edited (I11 + I13).
func (s *Submission) CheckCanEdit(deadlineAt *time.Time) error {
	if s.Status != StatusDraft {
		return fmt.Errorf("%w", ErrNotDraft)
	}
	if deadlineAt != nil && time.Now().UTC().After(*deadlineAt) {
		return fmt.Errorf("%w: deadline was %s", ErrDeadlinePassed, deadlineAt.Format(time.RFC3339))
	}
	return nil
}

// Submit transitions a draft submission to submitted (I13).
func (s *Submission) Submit(deadlineAt *time.Time) error {
	if err := s.CheckCanEdit(deadlineAt); err != nil {
		return err
	}
	now := time.Now().UTC()
	s.Status = StatusSubmitted
	s.SubmittedAt = &now
	s.UpdatedAt = now
	return nil
}
