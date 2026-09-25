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
