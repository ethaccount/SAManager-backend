package domain

import (
	"encoding/json"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// Credential is our GORM-friendly version of webauthn.Credential
type Credential struct {
	ID              []byte                   `gorm:"primaryKey;type:bytea"`
	UserID          []byte                   `gorm:"not null;type:bytea"`
	PublicKey       []byte                   `gorm:"not null;type:bytea"`
	AttestationType string                   `gorm:"not null"`
	Transports      []byte                   `gorm:"type:bytea"` // JSON encoded protocol.AuthenticatorTransport
	Flags           webauthn.CredentialFlags `gorm:"type:integer"`
	Authenticator   []byte                   `gorm:"type:bytea"` // JSON encoded authenticator data
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ToWebauthn converts our GORM model to webauthn.Credential
func (c *Credential) ToWebauthn() (*webauthn.Credential, error) {
	var transports []protocol.AuthenticatorTransport
	if err := json.Unmarshal(c.Transports, &transports); err != nil {
		return nil, err
	}

	var authenticator webauthn.Authenticator
	if err := json.Unmarshal(c.Authenticator, &authenticator); err != nil {
		return nil, err
	}

	return &webauthn.Credential{
		ID:              c.ID,
		PublicKey:       c.PublicKey,
		AttestationType: c.AttestationType,
		Transport:       transports,
		Flags:           c.Flags,
		Authenticator:   authenticator,
	}, nil
}

// FromWebauthn converts webauthn.Credential to our GORM model
func (c *Credential) FromWebauthn(wc *webauthn.Credential, userID []byte) error {
	transports, err := json.Marshal(wc.Transport)
	if err != nil {
		return err
	}

	authenticator, err := json.Marshal(wc.Authenticator)
	if err != nil {
		return err
	}

	c.ID = wc.ID
	c.UserID = userID
	c.PublicKey = wc.PublicKey
	c.AttestationType = wc.AttestationType
	c.Transports = transports
	c.Flags = wc.Flags
	c.Authenticator = authenticator

	return nil
}
