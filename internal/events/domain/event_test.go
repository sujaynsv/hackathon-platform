package domain_test

import (
	"testing"

	"github.com/dogfood-platform/dogfood/internal/events/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestEvent_Transition_ValidPath_Succeeds(t *testing.T) {
	e, err := domain.NewEvent("test-slug", "Test Event", uuid.New(), 5)
	assert.NoError(t, err)
	assert.Equal(t, domain.StatusDraft, e.Status)

	err = e.Transition(domain.StatusRegistrationOpen)
	assert.NoError(t, err)
	assert.Equal(t, domain.StatusRegistrationOpen, e.Status)
}

func TestEvent_Transition_InvalidSkip_ReturnsErrInvalidTransition(t *testing.T) {
	e, err := domain.NewEvent("test-slug", "Test Event", uuid.New(), 5)
	assert.NoError(t, err)
	assert.Equal(t, domain.StatusDraft, e.Status)

	err = e.Transition(domain.StatusJudging)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	assert.Equal(t, domain.StatusDraft, e.Status) // state unchanged
}

func TestEvent_Transition_Backward_ReturnsErrInvalidTransition(t *testing.T) {
	e, err := domain.NewEvent("test-slug", "Test Event", uuid.New(), 5)
	assert.NoError(t, err)
	e.Status = domain.StatusRegistrationOpen // force state for test

	err = e.Transition(domain.StatusDraft)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidTransition)
}

func TestNewEvent_EmptySlug_ReturnsError(t *testing.T) {
	e, err := domain.NewEvent("", "Test Event", uuid.New(), 5)
	assert.Error(t, err)
	assert.Nil(t, e)
	assert.Equal(t, "slug is required", err.Error())
}

func TestNewEvent_EmptyTitle_ReturnsError(t *testing.T) {
	e, err := domain.NewEvent("slug", "", uuid.New(), 5)
	assert.Error(t, err)
	assert.Nil(t, e)
	assert.Equal(t, "title is required", err.Error())
}
