package repository

import (
	"github.com/ethaccount/backend/src/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PasskeyRepository struct {
	db *gorm.DB
}

func NewPasskeyRepository(db *gorm.DB) *PasskeyRepository {
	return &PasskeyRepository{db: db}
}

func (r *PasskeyRepository) GetOrCreateUser(username string) (domain.User, error) {
	var u domain.User

	// Generate a UUID for new users
	id := uuid.New().String()

	result := r.db.Preload("Credentials").Where("name = ?", username).FirstOrCreate(&u, domain.User{
		ID:   []byte(id),
		Name: username,
	})

	return u, result.Error
}

func (r *PasskeyRepository) SaveCredential(userID []byte, cred *domain.Credential) error {
	return r.db.Create(cred).Error
}
