package domain

import (
	"errors"
	"fmt"
	"time"
)

var ErrEventNotOpen = errors.New("event is not open for registration")
var ErrDeadlinePassed = errors.New("registration deadline has passed")
var ErrAlreadyHasTeam = errors.New("cannot unregister: user already has a team in this event")

func CheckCanRegister(eventStatus string, registrationClosesAt *time.Time) error {
	if eventStatus != "registration_open" {
		return fmt.Errorf("%w: event status is '%s'", ErrEventNotOpen, eventStatus)
	}
	if registrationClosesAt != nil && time.Now().UTC().After(*registrationClosesAt) {
		return fmt.Errorf("%w: registration closed at %s", ErrDeadlinePassed, registrationClosesAt.Format(time.RFC3339))
	}
	return nil
}
