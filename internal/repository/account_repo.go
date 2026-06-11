package repository

import (
	"exotel-monitoring-platform/internal/database"
	"exotel-monitoring-platform/internal/models"
)

// GetActiveAccounts returns all non-deleted active accounts.
func GetActiveAccounts() ([]models.Account, error) {
	var accounts []models.Account
	result := database.DB.
		Where("is_active = 1 AND is_deleted = 0").
		Find(&accounts)
	return accounts, result.Error
}

// GetAccountByID retrieves an account by primary key.
func GetAccountByID(id uint64) (*models.Account, error) {
	var account models.Account
	result := database.DB.
		Where("id = ? AND is_deleted = 0", id).
		First(&account)
	return &account, result.Error
}

// GetAccountBySID retrieves an account by its Exotel SID.
func GetAccountBySID(sid string) (*models.Account, error) {
	var account models.Account
	result := database.DB.
		Where("sid = ? AND is_deleted = 0", sid).
		First(&account)
	return &account, result.Error
}

// UpsertAccount creates or updates an account.
func UpsertAccount(a *models.Account) error {
	return database.DB.Save(a).Error
}
