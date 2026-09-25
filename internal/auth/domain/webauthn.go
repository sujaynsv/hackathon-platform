package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type WebAuthnCredential struct {
	ID                  uuid.UUID
	UserID              uuid.UUID
	CredentialID        []byte
	PublicKey           []byte
	AttestationType     string
	Transport           []string // e.g. ["usb", "nfc", "ble", "internal"]
	Flags               CredentialFlags
	AuthenticatorAAGUID []byte
	SignCount           uint32
	CloneWarning        bool
	CreatedAt           time.Time
	LastUsedAt          *time.Time
}

type CredentialFlags struct {
	UserPresent    bool `json:"userPresent"`
	UserVerified   bool `json:"userVerified"`
	BackupEligible bool `json:"backupEligible"`
	BackupState    bool `json:"backupState"`
}

func (f *CredentialFlags) ScanJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, f)
}

func (f *CredentialFlags) ToJSON() ([]byte, error) {
	return json.Marshal(f)
}
