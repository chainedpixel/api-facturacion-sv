package drivers

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MysqlDriver struct{}

func NewMysqlDriver() *MysqlDriver {
	return &MysqlDriver{}
}

func (m *MysqlDriver) GetDSN() gorm.Dialector {
	return mysql.Open(m.GetStringConnection())
}

func (m *MysqlDriver) GetStringConnection() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=America%%2FEl_Salvador",
		config.Database.User,
		config.Database.Password,
		config.Database.Host,
		config.Database.Port,
		config.Database.Name,
		config.Database.Charset,
	)
}

func (m *MysqlDriver) GetHost() string {
	return config.Database.Port
}

func (m *MysqlDriver) GetDriverName() string {
	return "MySQL"
}
