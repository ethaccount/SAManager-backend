package domain

import (
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

// User implements the webauthn.User interface
type User struct {
	ID          []byte       `gorm:"primaryKey;type:bytea"`
	Name        string       `gorm:"unique;not null"`
	Credentials []Credential `gorm:"foreignKey:UserID"`
	CreatedAt   time.Time
	UpdatedAt   time.Time

	// Cache for webauthn credentials
	webauthnCredentials []webauthn.Credential
}

func (u *User) WebAuthnID() []byte {
	return u.ID
}

func (u *User) WebAuthnName() string {
	return u.Name
}

func (u *User) WebAuthnDisplayName() string {
	return u.Name
}

func (u *User) WebAuthnCredentials() []webauthn.Credential {
	// Convert stored credentials to webauthn credentials if not cached
	if u.webauthnCredentials == nil {
		u.webauthnCredentials = make([]webauthn.Credential, 0, len(u.Credentials))
		for _, cred := range u.Credentials {
			if wc, err := cred.ToWebauthn(); err == nil {
				u.webauthnCredentials = append(u.webauthnCredentials, *wc)
			}
		}
	}
	return u.webauthnCredentials
}

func (u *User) AddCredential(cred webauthn.Credential) error {
	var c Credential
	if err := c.FromWebauthn(&cred, u.ID); err != nil {
		return err
	}
	u.Credentials = append(u.Credentials, c)
	// Clear the cache
	u.webauthnCredentials = nil
	return nil
}
