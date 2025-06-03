package service

import (
	"context"
	"time"

	"github.com/ethaccount/backend/src/repository"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/rs/zerolog"
)

type PasskeyService struct {
	repo     *repository.PasskeyRepository
	webauthn *webauthn.WebAuthn
	ttl      time.Duration
}

func NewPasskeyService(_ context.Context, repo *repository.PasskeyRepository, config *webauthn.Config, ttl time.Duration) (*PasskeyService, error) {
	w, err := webauthn.New(config)
	if err != nil {
		return nil, err
	}

	return &PasskeyService{
		repo:     repo,
		webauthn: w,
		ttl:      ttl,
	}, nil
}

// logger wrap the execution context with component info
func (s *PasskeyService) logger(ctx context.Context) *zerolog.Logger {
	l := zerolog.Ctx(ctx).With().Str("service", "passkey").Logger()
	return &l
}

func (s *PasskeyService) BeginRegistration(ctx context.Context, username string) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
	s.logger(ctx).Info().Msgf("BeginRegistration: %s", username)

	// Get or create user
	user, err := s.repo.GetOrCreateUser(username)
	if err != nil {
		s.logger(ctx).Error().Err(err).Msg("failed to get or create user")
		return nil, nil, err
	}

	// Begin registration
	options, session, err := s.webauthn.BeginRegistration(&user)
	if err != nil {
		s.logger(ctx).Error().Err(err).Msg("failed to begin registration")
		return nil, nil, err
	}

	s.logger(ctx).Debug().
		Str("rpID", s.webauthn.Config.RPID).
		Str("rpName", s.webauthn.Config.RPDisplayName).
		Str("username", user.Name).
		Str("challenge", string(session.Challenge)).
		Msg("registration options created")

	return options, session, nil
}
