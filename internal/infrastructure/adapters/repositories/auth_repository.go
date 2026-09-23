package repositories

import (
	"context"
	"errors"

	"github.com/chainedpixel/ordo-factus/internal/domain/auth"
	"gorm.io/gorm"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/user"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/database/db_models"
	errPackage "github.com/chainedpixel/ordo-factus/internal/infrastructure/error"
)

type AuthRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) auth.AuthRepositoryPort {
	return &AuthRepository{db: db}
}

// GetAuthTypeByApiKey retrieves the authentication type of a user by their API key
func (r *AuthRepository) GetAuthTypeByApiKey(ctx context.Context, apiKey string) (string, error) {
	user, err := r.GetByBranchApiKey(ctx, apiKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errPackage.ErrUserNotFound
		}
	}

	if user == nil {
		return "", errPackage.ErrUserNotFound
	}

	return user.AuthType, nil
}

// GetByNIT retrieves a user by their NIT
func (r *AuthRepository) GetByNIT(ctx context.Context, nit string) (*user.User, error) {
	var dbUser db_models.User
	result := r.db.WithContext(ctx).Where("nit = ? AND status = ?", nit, true).First(&dbUser)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errPackage.ErrUserNotFound
		}
		return nil, result.Error
	}

	user := &user.User{
		ID:             dbUser.ID,
		NIT:            dbUser.NIT,
		NRC:            dbUser.NRC,
		Status:         dbUser.Status,
		AuthType:       dbUser.AuthType,
		PasswordPri:    dbUser.PasswordPri,
		CommercialName: dbUser.CommercialName,
		Business:       dbUser.Business,
		Email:          dbUser.Email,
		YearInDTE:      dbUser.YearInDTE,
		Phone:          dbUser.Phone,
		TokenLifetime:  dbUser.TokenLifetime,
		CreatedAt:      dbUser.CreatedAt,
		UpdatedAt:      dbUser.UpdatedAt,
	}

	return user, nil
}

// GetByBranchApiKey retrieves a user by the API key of a branch office
func (r *AuthRepository) GetByBranchApiKey(ctx context.Context, apiKey string) (*user.User, error) {
	var branch db_models.BranchOffice

	result := r.db.WithContext(ctx).Where("api_key = ? AND is_active = ?", apiKey, true).First(&branch)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errPackage.ErrBranchOfficeNotFound
		}
		return nil, result.Error
	}

	var dbUser db_models.User
	result = r.db.WithContext(ctx).Where("id = ? AND status = ?", branch.UserID, true).First(&dbUser)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errPackage.ErrUserNotFound
		}
		return nil, result.Error
	}

	return &user.User{
		ID:                   dbUser.ID,
		NIT:                  dbUser.NIT,
		NRC:                  dbUser.NRC,
		Status:               dbUser.Status,
		AuthType:             dbUser.AuthType,
		PasswordPri:          dbUser.PasswordPri,
		CommercialName:       dbUser.CommercialName,
		EconomicActivity:     dbUser.EconomicActivity,
		EconomicActivityDesc: dbUser.EconomicActivityDesc,
		Phone:                dbUser.Phone,
		Business:             dbUser.Business,
		Email:                dbUser.Email,
		TokenLifetime:        dbUser.TokenLifetime,
		YearInDTE:            dbUser.YearInDTE,
		CreatedAt:            dbUser.CreatedAt,
		UpdatedAt:            dbUser.UpdatedAt,
	}, nil
}

// GetBranchByBranchApiKey retrieves a branch office by its API key
func (r *AuthRepository) GetBranchByBranchApiKey(ctx context.Context, apiKey string) (*user.BranchOffice, error) {
	var branch db_models.BranchOffice

	result := r.db.WithContext(ctx).Preload("Address").Where("api_key = ?", apiKey).First(&branch)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errPackage.ErrBranchOfficeNotFound
		}
		return nil, result.Error
	}

	localUser := &user.BranchOffice{
		ID:                  branch.ID,
		UserID:              branch.UserID,
		EstablishmentCode:   branch.EstablishmentCode,
		EstablishmentCodeMH: branch.EstablishmentCodeMH,
		Email:               branch.Email,
		APIKey:              branch.APIKey,
		APISecret:           branch.APISecret,
		Phone:               branch.Phone,
		EstablishmentType:   branch.EstablishmentType,
		POSCode:             branch.POSCode,
		POSCodeMH:           branch.POSCodeMH,
		IsActive:            branch.IsActive,
	}

	if localUser.Address != nil {
		localUser.Address = &user.Address{
			Municipality: branch.Address.Municipality,
			Department:   branch.Address.Department,
			District:     branch.Address.District,
			Complement:   branch.Address.Complement,
		}
	}

	return localUser, nil
}

// GetByBranchID retrieves a user by the ID of a branch office
func (r *AuthRepository) GetByBranchID(ctx context.Context, branchID uint) (*user.User, error) {
	var branch db_models.BranchOffice

	result := r.db.WithContext(ctx).Where("id = ? AND is_active = ?", branchID, true).First(&branch)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errPackage.ErrBranchOfficeNotFound
		}
		return nil, result.Error
	}

	var dbUser db_models.User
	result = r.db.WithContext(ctx).Where("id = ? AND status = ?", branch.UserID, true).First(&dbUser)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errPackage.ErrUserNotFound
		}
		return nil, result.Error
	}

	return &user.User{
		ID:                   dbUser.ID,
		NIT:                  dbUser.NIT,
		NRC:                  dbUser.NRC,
		Status:               dbUser.Status,
		AuthType:             dbUser.AuthType,
		PasswordPri:          dbUser.PasswordPri,
		CommercialName:       dbUser.CommercialName,
		EconomicActivity:     dbUser.EconomicActivity,
		EconomicActivityDesc: dbUser.EconomicActivityDesc,
		Phone:                dbUser.Phone,
		Business:             dbUser.Business,
		Email:                dbUser.Email,
		TokenLifetime:        dbUser.TokenLifetime,
		YearInDTE:            dbUser.YearInDTE,
		CreatedAt:            dbUser.CreatedAt,
		UpdatedAt:            dbUser.UpdatedAt,
	}, nil
}

// Create creates a user along with their branch offices
func (r *AuthRepository) Create(ctx context.Context, user *user.User) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dbUser := db_models.User{
			NIT:                  user.NIT,
			NRC:                  user.NRC,
			Status:               true,
			AuthType:             user.AuthType,
			PasswordPri:          user.PasswordPri,
			CommercialName:       user.CommercialName,
			EconomicActivity:     user.EconomicActivity,
			EconomicActivityDesc: user.EconomicActivityDesc,
			Business:             user.Business,
			Email:                user.Email,
			Phone:                user.Phone,
			YearInDTE:            user.YearInDTE,
		}

		if user.TokenLifetime != 0 {
			dbUser.TokenLifetime = user.TokenLifetime
		}

		if err := tx.Create(&dbUser).Error; err != nil {
			return err
		}

		user.ID = dbUser.ID

		for i := range user.BranchOffices {
			dbBranch := db_models.BranchOffice{
				UserID:              dbUser.ID,
				EstablishmentCode:   user.BranchOffices[i].EstablishmentCode,
				EstablishmentCodeMH: user.BranchOffices[i].EstablishmentCodeMH,
				Email:               user.BranchOffices[i].Email,
				APIKey:              user.BranchOffices[i].APIKey,
				APISecret:           user.BranchOffices[i].APISecret,
				Phone:               user.BranchOffices[i].Phone,
				EstablishmentType:   user.BranchOffices[i].EstablishmentType,
				POSCode:             user.BranchOffices[i].POSCode,
				POSCodeMH:           user.BranchOffices[i].POSCodeMH,
				IsActive:            user.BranchOffices[i].IsActive,
			}

			if err := tx.Create(&dbBranch).Error; err != nil {
				return err
			}

			if user.BranchOffices[i].Address != nil {
				dbAddress := db_models.Address{
					BranchID:     dbBranch.ID,
					Municipality: user.BranchOffices[i].Address.Municipality,
					Department:   user.BranchOffices[i].Address.Department,
					District:     user.BranchOffices[i].Address.District,
					Complement:   user.BranchOffices[i].Address.Complement,
				}

				if err := tx.Create(&dbAddress).Error; err != nil {
					return err
				}
			}

			user.BranchOffices[i].ID = dbBranch.ID
		}

		return nil
	})
}

// Update updates a user
func (r *AuthRepository) Update(ctx context.Context, user *user.User) error {
	dbUser := db_models.User{
		ID:             user.ID,
		NIT:            user.NIT,
		NRC:            user.NRC,
		Status:         user.Status,
		AuthType:       user.AuthType,
		PasswordPri:    user.PasswordPri,
		CommercialName: user.CommercialName,
		Business:       user.Business,
		Email:          user.Email,
		YearInDTE:      user.YearInDTE,
		Phone:          user.Phone,
		TokenLifetime:  user.TokenLifetime,
	}

	return r.db.WithContext(ctx).Model(&dbUser).Updates(dbUser).Error
}

// UpdateBranchOffices updates the branch offices of a user
func (r *AuthRepository) UpdateBranchOffices(ctx context.Context, userID uint, branches []user.BranchOffice) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, branch := range branches {
			var count int64
			tx.Model(&db_models.BranchOffice{}).Where("id = ? AND user_id = ?", branch.ID, userID).Count(&count)
			if count == 0 {
				return errPackage.ErrBranchDoesNotBelong
			}

			dbBranch := db_models.BranchOffice{
				ID:                  branch.ID,
				EstablishmentCode:   branch.EstablishmentCode,
				EstablishmentCodeMH: branch.EstablishmentCodeMH,
				Email:               branch.Email,
				APIKey:              branch.APIKey,
				APISecret:           branch.APISecret,
				Phone:               branch.Phone,
				EstablishmentType:   branch.EstablishmentType,
				POSCode:             branch.POSCode,
				POSCodeMH:           branch.POSCodeMH,
				IsActive:            branch.IsActive,
			}

			if err := tx.Model(&dbBranch).Updates(dbBranch).Error; err != nil {
				return err
			}

			if branch.Address != nil {
				dbAddress := db_models.Address{
					BranchID:     branch.ID,
					Municipality: branch.Address.Municipality,
					Department:   branch.Address.Department,
					District:     branch.Address.District,
					Complement:   branch.Address.Complement,
				}

				var existingAddress db_models.Address
				result := tx.Where("branch_id = ?", branch.ID).First(&existingAddress)
				if result.Error != nil {
					if errors.Is(result.Error, gorm.ErrRecordNotFound) {
						if err := tx.Create(&dbAddress).Error; err != nil {
							return err
						}
					} else {
						return result.Error
					}
				} else {
					dbAddress.ID = existingAddress.ID
					if err := tx.Model(&dbAddress).Updates(dbAddress).Error; err != nil {
						return err
					}
				}
			}
		}

		return nil
	})
}

// DeleteBranchOffice deletes a branch office from a user
func (r *AuthRepository) DeleteBranchOffice(ctx context.Context, userID uint, branchID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		tx.Model(&db_models.BranchOffice{}).Where("id = ? AND user_id = ?", branchID, userID).Count(&count)
		if count == 0 {
			return errPackage.ErrBranchDoesNotBelong
		}

		if err := tx.Where("branch_id = ?", branchID).Delete(&db_models.Address{}).Error; err != nil {
			return err
		}

		return tx.Delete(&db_models.BranchOffice{}, branchID).Error
	})
}

func (r *AuthRepository) GetBranchByBranchID(ctx context.Context, branchID uint) (*user.BranchOffice, error) {
	var branch db_models.BranchOffice

	result := r.db.WithContext(ctx).
		Preload("Address").
		Preload("User").
		Where("id = ?", branchID).First(&branch)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errPackage.ErrBranchOfficeNotFound
		}
		return nil, result.Error
	}

	localBranch := &user.BranchOffice{
		ID:                  branch.ID,
		UserID:              branch.UserID,
		EstablishmentCode:   branch.EstablishmentCode,
		EstablishmentCodeMH: branch.EstablishmentCodeMH,
		Email:               branch.Email,
		APIKey:              branch.APIKey,
		APISecret:           branch.APISecret,
		Phone:               branch.Phone,
		EstablishmentType:   branch.EstablishmentType,
		POSCode:             branch.POSCode,
		POSCodeMH:           branch.POSCodeMH,
		IsActive:            branch.IsActive,
		User: &user.User{
			ID:                   branch.User.ID,
			Status:               branch.User.Status,
			Email:                branch.User.Email,
			Phone:                branch.User.Phone,
			NIT:                  branch.User.NIT,
			NRC:                  branch.User.NRC,
			AuthType:             branch.User.AuthType,
			EconomicActivity:     branch.User.EconomicActivity,
			EconomicActivityDesc: branch.User.EconomicActivityDesc,
		},
	}

	if localBranch.Address != nil {
		localBranch.Address = &user.Address{
			Municipality: branch.Address.Municipality,
			Department:   branch.Address.Department,
			District:     branch.Address.District,
			Complement:   branch.Address.Complement,
		}
	}

	return localBranch, nil
}

// GetAuthTypeByNIT retrieves the authentication type of a user by their NIT
func (r *AuthRepository) GetAuthTypeByNIT(ctx context.Context, nit string) (string, error) {
	user, err := r.GetByNIT(ctx, nit)
	if err != nil {
		return "", err
	}

	return user.AuthType, nil
}

// GetIssuerInfoByBranchID retrieves the user and branch information formatted for the Hacienda DTE
func (r *AuthRepository) GetIssuerInfoByBranchID(ctx context.Context, branchID uint) (*dte.IssuerDTE, error) {
	branch, err := r.GetBranchByBranchID(ctx, branchID)
	if err != nil {
		return nil, err
	}
	phone := branch.Phone
	email := branch.Email

	user, err := r.GetByBranchID(ctx, branchID)
	if err != nil {
		return nil, err
	}

	if branch.Email == nil {
		email = &user.Email
	}

	if branch.Phone == nil {
		phone = &user.Phone
	}

	if branch.Address == nil {
		matrixBranch, err := r.GetMatrixBranch(ctx, user.ID)
		if err != nil {
			return nil, err
		}

		branch.Address = matrixBranch.Address
	}

	if branch.POSCode != nil && len(*branch.POSCode) > 4 {
		*branch.POSCode = (*branch.POSCode)[:4]
	}

	if branch.EstablishmentCode != nil && len(*branch.EstablishmentCode) > 4 {
		*branch.EstablishmentCode = (*branch.EstablishmentCode)[:4]
	}

	return &dte.IssuerDTE{
		NIT:                  user.NIT,
		NRC:                  user.NRC,
		CommercialName:       user.CommercialName,
		BusinessName:         user.Business,
		EconomicActivity:     user.EconomicActivity,
		EconomicActivityDesc: user.EconomicActivityDesc,
		EstablishmentCode:    branch.EstablishmentCode,
		EstablishmentCodeMH:  branch.EstablishmentCodeMH,
		EstablishmentType:    branch.EstablishmentType,
		POSCode:              branch.POSCode,
		POSCodeMH:            branch.POSCodeMH,
		Email:                email,
		Phone:                phone,
		Address:              branch.Address,
	}, nil
}

// GetMatrixBranch retrieves the branch office registered as the main office (casa matriz)
func (r *AuthRepository) GetMatrixBranch(ctx context.Context, userID uint) (*user.BranchOffice, error) {
	var branch db_models.BranchOffice

	result := r.db.WithContext(ctx).
		Preload("Address").
		Where("user_id = ? AND establishment_type = ?", userID, constants.CasaMatriz).
		First(&branch)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errPackage.ErrBranchOfficeNotFound
		}
		return nil, result.Error
	}

	return &user.BranchOffice{
		ID:                  branch.ID,
		UserID:              branch.UserID,
		EstablishmentCode:   branch.EstablishmentCode,
		EstablishmentCodeMH: branch.EstablishmentCodeMH,
		Email:               branch.Email,
		APIKey:              branch.APIKey,
		APISecret:           branch.APISecret,
		Phone:               branch.Phone,
		EstablishmentType:   branch.EstablishmentType,
		POSCode:             branch.POSCode,
		POSCodeMH:           branch.POSCodeMH,
		IsActive:            branch.IsActive,
		Address: &user.Address{
			Municipality: branch.Address.Municipality,
			Department:   branch.Address.Department,
			District:     branch.Address.District,
			Complement:   branch.Address.Complement,
		},
	}, nil
}
