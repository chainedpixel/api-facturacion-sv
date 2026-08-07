package db_models

import "time"

// User represents the table whose information will be crucial for the authentication and authorization process of users
// who wish to use the Electronic Invoice API. This table will store the information of users who want
// to use the Electronic Invoice API.
//
// The PasswordPri field holds the private password of the user's digital certificate provided by the Ministry of Finance,
// this password is required to sign electronic documents.
//
// The YearInDTE field is an indicator to determine whether the user wants the year of issuance of the electronic document
// to appear in the control number. For example:
//  1. If true, the control number will be: DTE-01-00000000-202500000000001
//  2. If false, the control number will be: DTE-01-00000000-000000000000001
type User struct {
	ID                   uint      `gorm:"column:id;type:uint;primaryKey;autoIncrement;not null"`
	NIT                  string    `gorm:"column:nit;type:varchar(17);not null;uniqueIndex"`
	NRC                  string    `gorm:"column:nrc;type:varchar(10);not null;uniqueIndex"`
	Status               bool      `gorm:"column:status;type:tinyint;not null;index:idx_user_status"`
	AuthType             string    `gorm:"column:auth_type;type:varchar(15);not null"`
	PasswordPri          string    `gorm:"column:password_pri;type:varchar(255);not null"`
	CommercialName       string    `gorm:"column:commercial_name;type:varchar(150);not null"`
	EconomicActivity     string    `gorm:"column:economic_activity;type:varchar(6);not null"`
	EconomicActivityDesc string    `gorm:"column:economic_activity_desc;type:varchar(150);not null"`
	Business             string    `gorm:"column:business_name;type:varchar(200);not null"`
	Email                string    `gorm:"column:email;type:varchar(100);not null;uniqueIndex:idx_user_email"`
	Phone                string    `gorm:"column:phone;type:varchar(30);not null;uniqueIndex:idx_user_phone"`
	YearInDTE            bool      `gorm:"column:year_in_dte;type:tinyint;not null"`
	TokenLifetime        int       `gorm:"column:token_lifetime;type:int;not null;default:14"`
	CreatedAt            time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP"`
	UpdatedAt            time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP"`
}

func (User) TableName() string {
	return "users"
}
