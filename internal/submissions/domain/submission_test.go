package domain_test

import (
	"testing"

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
