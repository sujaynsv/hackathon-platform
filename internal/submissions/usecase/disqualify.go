package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/submissions/domain"
	"github.com/dogfood-platform/dogfood/internal/submissions/port"
)

type DisqualifyService struct {
	subs     port.SubmissionRepository
	users    port.UserReader
	auditLog port.AuditLogWriter
}

func NewDisqualifyService(
	subs port.SubmissionRepository,
	users port.UserReader,
	auditLog port.AuditLogWriter,
) *DisqualifyService {
	return &DisqualifyService{
		subs:     subs,
		users:    users,
		auditLog: auditLog,
	}
}

func (s *DisqualifyService) Disqualify(ctx context.Context, cmd port.DisqualifyCommand) error {
	// 1. Verify caller is admin (I16)
	isAdmin, err := s.users.IsAdmin(ctx, cmd.AdminID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("%w: only admins can disqualify submissions", response.ErrForbidden)
	}

	// 2. Load submission
	sub, err := s.subs.FindByID(ctx, cmd.SubmissionID)
	if err != nil {
		return err
	}
	if sub == nil {
		return fmt.Errorf("%w: submission not found", response.ErrNotFound)
	}

	// 3. Disqualify
	sub.Status = domain.StatusDisqualified
	sub.UpdatedAt = time.Now().UTC()
	if err := s.subs.Update(ctx, sub); err != nil {
		return err
	}

	// 4. I17: always write audit log
	return s.auditLog.Write(ctx, &port.AuditEntry{
		ActorID:      cmd.AdminID,
		Action:       "DISQUALIFY_SUBMISSION",
		ResourceType: "submission",
		ResourceID:   sub.ID,
		Changes: map[string]any{
			"reason": cmd.Reason,
			"status": "disqualified",
		},
	})
}
