package domain_test

import (
	"testing"

	"github.com/dogfood-platform/dogfood/internal/submissions/domain"
	"github.com/stretchr/testify/assert"
)

func TestValidateUpload_ValidCoverJPEG_ReturnsNil(t *testing.T) {
	err := domain.ValidateUpload("image/jpeg", "cover", 1024)
	assert.NoError(t, err)
}

func TestValidateUpload_AttachmentPDF_ReturnsNil(t *testing.T) {
	err := domain.ValidateUpload("application/pdf", "attachment", 1024)
	assert.NoError(t, err)
}

func TestValidateUpload_CoverPDF_ReturnsErrInvalidMIMEType(t *testing.T) {
	err := domain.ValidateUpload("application/pdf", "cover", 1024)
	assert.ErrorIs(t, err, domain.ErrInvalidMIMEType)
}

func TestValidateUpload_ExceedsMaxSize_ReturnsErrFileTooLarge(t *testing.T) {
	err := domain.ValidateUpload("image/jpeg", "cover", domain.MaxFileSizeBytes+1)
	assert.ErrorIs(t, err, domain.ErrFileTooLarge)
}
