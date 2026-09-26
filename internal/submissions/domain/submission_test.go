package domain_test

import (
	"testing"
	"time"

	"github.com/dogfood-platform/dogfood/internal/submissions/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewDraftSubmission(t *testing.T) {
	teamID := uuid.New()
	eventID := uuid.New()
	title := "My Hackathon Project"

	sub, err := domain.NewDraftSubmission(teamID, eventID, nil, title)
	assert.NoError(t, err)
	assert.NotNil(t, sub)
	assert.Equal(t, teamID, sub.TeamID)
	assert.Equal(t, eventID, sub.EventID)
	assert.Equal(t, title, sub.Title)
	assert.Equal(t, domain.StatusDraft, sub.Status)
}

func TestNewDraftSubmission_EmptyTitle(t *testing.T) {
	teamID := uuid.New()
	eventID := uuid.New()

	sub, err := domain.NewDraftSubmission(teamID, eventID, nil, "")
	assert.Error(t, err)
	assert.Nil(t, sub)
	assert.Equal(t, "title is required", err.Error())
}

func TestCheckCanCreate(t *testing.T) {
	err := domain.CheckCanCreate("submissions_open")
	assert.NoError(t, err)

	err = domain.CheckCanCreate("draft")
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrSubmissionsNotOpen)
}

func TestSubmission_CheckCanEdit_DraftBeforeDeadline_Nil(t *testing.T) {
	sub, _ := domain.NewDraftSubmission(uuid.New(), uuid.New(), nil, "Title")
	// No deadline
	err := sub.CheckCanEdit(nil)
	assert.NoError(t, err)

	// Future deadline
	future := time.Now().UTC().Add(time.Hour)
	err = sub.CheckCanEdit(&future)
	assert.NoError(t, err)
}

func TestSubmission_CheckCanEdit_AfterDeadline_ErrDeadlinePassed(t *testing.T) {
	sub, _ := domain.NewDraftSubmission(uuid.New(), uuid.New(), nil, "Title")
	past := time.Now().UTC().Add(-time.Hour)
	err := sub.CheckCanEdit(&past)
	assert.ErrorIs(t, err, domain.ErrDeadlinePassed)
}

func TestSubmission_CheckCanEdit_AlreadySubmitted_ErrNotDraft(t *testing.T) {
	sub, _ := domain.NewDraftSubmission(uuid.New(), uuid.New(), nil, "Title")
	sub.Status = domain.StatusSubmitted
	err := sub.CheckCanEdit(nil)
	assert.ErrorIs(t, err, domain.ErrNotDraft)
}

func TestSubmission_Submit_DraftBeforeDeadline_SetsStatusAndTimestamp(t *testing.T) {
	sub, _ := domain.NewDraftSubmission(uuid.New(), uuid.New(), nil, "Title")
	err := sub.Submit(nil)
	assert.NoError(t, err)
	assert.Equal(t, domain.StatusSubmitted, sub.Status)
	assert.NotNil(t, sub.SubmittedAt)
}

func TestSubmission_Submit_AlreadySubmitted_ReturnsError(t *testing.T) {
	sub, _ := domain.NewDraftSubmission(uuid.New(), uuid.New(), nil, "Title")
	sub.Status = domain.StatusSubmitted
	err := sub.Submit(nil)
	assert.ErrorIs(t, err, domain.ErrNotDraft)
}
