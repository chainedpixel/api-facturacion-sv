package config

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	errPackage "github.com/chainedpixel/ordo-factus/config/error"
	"github.com/spf13/viper"
)

var (
	URLPattern  = "https?:\\/\\/(?:localhost|(?:www\\.)?[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}|(?:\\d{1,3}\\.){3}\\d{1,3})(:\\d{1,5})?(\\/\\S*)?"
	PortPattern = "\\b(0|[1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])\\b"
	HostPattern = "^(localhost|((25[0-5]|2[0-4]\\d|[0-1]?\\d?\\d)\\.){3}(25[0-5]|2[0-4]\\d|[0-1]?\\d?\\d)|((?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?\\.)+[a-zA-Z]{2,}))$"
)

var (
	AvailableDatabaseDrivers = map[string]bool{
		"mysql":    true,
		"postgres": true,
	}
)

var EnvConfig *envConfig
var Server *server
var Database *database
var Redis *redis
var Log *log
var Signer *signer
var MHPaths *mhPaths
var SMTP *smtp

// InitEnvTesting initializes the test environment configuration
func InitEnvTesting() {
	EnvConfig = &envConfig{}
	Server = &EnvConfig.Server
	Database = &EnvConfig.Database
	Redis = &EnvConfig.Redis
	Log = &EnvConfig.Log
	Signer = &EnvConfig.Signer
	MHPaths = &EnvConfig.MHPaths
	SMTP = &EnvConfig.SMTP

	Server.AmbientCode = "00"
	Log.Path = "/assets/logs/"
	Server.Debug = true
	Server.AppLang = "en"
	Server.AppLangPath = "/assets/i18n/"
	Server.APIVersion = "3.0.0"
	Server.NotifyCooldownMinutes = 15
}

// InitEnvConfig initializes the configuration from the .env file
func InitEnvConfig(rootPath string) error {
	v := viper.New()
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(rootPath)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return errPackage.ErrEnvFileNotFound
		}
		return err
	}

	v.AutomaticEnv()
	EnvConfig = &envConfig{}

	if err := autoMapEnvKeys(v, reflect.ValueOf(EnvConfig).Elem()); err != nil {
		return err
	}

	if err := v.Unmarshal(&EnvConfig); err != nil {
		return errPackage.ErrFailedToLoadEnv
	}

	if err := ValidateConfig(); err != nil {
		return err
	}

	Server = &EnvConfig.Server
	Database = &EnvConfig.Database
	Redis = &EnvConfig.Redis
	Log = &EnvConfig.Log
	Signer = &EnvConfig.Signer
	MHPaths = &EnvConfig.MHPaths
	SMTP = &EnvConfig.SMTP

	if Server.APIVersion == "" {
		Server.APIVersion = "3.0.0"
	}
	if Server.NotifyCooldownMinutes <= 0 {
		Server.NotifyCooldownMinutes = 15
	}

	return nil
}

// ValidateConfig validates every field of the .env file configuration
func ValidateConfig() error {
	if err := validateServerFields(); err != nil {
		return err
	}

	if err := validateDatabaseFields(); err != nil {
		return err
	}

	if err := validateRedisFields(); err != nil {
		return err
	}

	if err := validateLogFields(); err != nil {
		return err
	}

	if err := validateMHConfigFields(); err != nil {
		return err
	}

	if err := validateSignerFields(); err != nil {
		return err
	}

	if err := validateSMTPFields(); err != nil {
		return err
	}

	return nil
}

// validateSMTPFields validates the optional SMTP struct
func validateSMTPFields() error {
	if EnvConfig.SMTP.Host == "" {
		return nil
	}

	if !matchPattern(HostPattern, EnvConfig.SMTP.Host) {
		return fmt.Errorf("SMTP_HOST must be a valid host")
	}
	if EnvConfig.SMTP.Port != "" && !matchPattern(PortPattern, EnvConfig.SMTP.Port) {
		return fmt.Errorf("SMTP_PORT must be a valid port")
	}
	if EnvConfig.SMTP.From == "" {
		return fmt.Errorf("SMTP_FROM is required when SMTP_HOST is set")
	}
	return nil
}

// validateServerFields validates the fields of the Server struct
func validateServerFields() error {
	bt := map[string]bool{
		"DEBUG":            true,
		"FORCECONTINGENCY": true,
		"RUNMIGRATION":     true,
	}
	ex := []string{"ADMINEMAIL", "APIVERSION", "NOTIFYCOOLDOWNMINUTES"}
	v := reflect.ValueOf(EnvConfig.Server)

	if err := validateEnvVariables(v, bt, ex); err != nil {
		return err
	}

	if !matchPattern(PortPattern, EnvConfig.Server.Port) {
		return fmt.Errorf("SERVER_PORT must be a valid port")
	}

	if EnvConfig.Server.MaxBatchSize <= 0 || EnvConfig.Server.MaxBatchSize > 100 {
		return fmt.Errorf("MH_MAX_BATCH_SIZE must be between 1 and 100")
	}

	return nil
}

// validateDatabaseFields validates the fields of the Database struct
func validateDatabaseFields() error {
	v := reflect.ValueOf(EnvConfig.Database)

	if err := validateEnvVariables(v, nil, nil); err != nil {
		return err
	}

	if !matchPattern(HostPattern, EnvConfig.Database.Host) {
		return fmt.Errorf("DATABASE_HOST must be a valid host")
	}

	if !matchPattern(PortPattern, EnvConfig.Database.Port) {
		return fmt.Errorf("DATABASE_PORT must be a valid port")
	}

	if !AvailableDatabaseDrivers[EnvConfig.Database.Driver] {
		return fmt.Errorf("DATABASE_DRIVER must be a valid driver")
	}

	return nil
}

// validateRedisFields validates the fields of the Redis struct
func validateRedisFields() error {
	v := reflect.ValueOf(EnvConfig.Redis)
	ex := []string{strings.ToUpper("PASSWORD")}

	if err := validateEnvVariables(v, nil, ex); err != nil {
		return err
	}

	if !matchPattern(HostPattern, EnvConfig.Redis.Host) {
		return fmt.Errorf("REDIS_HOST must be a valid host")
	}

	if !matchPattern(PortPattern, EnvConfig.Redis.Port) {
		return fmt.Errorf("REDIS_PORT must be a valid port")
	}

	return nil
}

// validateLogFields validates the fields of the Log struct
func validateLogFields() error {
	bt := map[string]bool{
		"FILELOGGING": true,
	}
	v := reflect.ValueOf(EnvConfig.Log)

	if err := validateEnvVariables(v, bt, nil); err != nil {
		return err
	}

	return nil
}

// validateMHConfigFields validates the fields of the MHPaths struct
func validateMHConfigFields() error {
	v := reflect.ValueOf(EnvConfig.MHPaths)

	if err := validateEnvVariables(v, nil, nil); err != nil {
		return err
	}

	for i := 0; i < v.NumField(); i++ {
		t := v.Type()
		f := v.Field(i)

		if !matchPattern(URLPattern, f.String()) {
			return fmt.Errorf("%s must be a valid URL", strings.ToUpper(t.Field(i).Name))
		}
	}

	return nil
}

// validateSignerFields validates the fields of the Signer struct
func validateSignerFields() error {
	v := reflect.ValueOf(EnvConfig.Signer)

	if err := validateEnvVariables(v, nil, nil); err != nil {
		return err
	}

	for i := 0; i < v.NumField(); i++ {
		t := v.Type()
		f := v.Field(i)

		if !matchPattern(URLPattern, f.String()) {
			return fmt.Errorf("%s must be a valid URL", strings.ToUpper(t.Field(i).Name))
		}
	}

	return nil
}

// validateEnvVariables validates that the struct fields are required and of the correct type
func validateEnvVariables(v reflect.Value, bt map[string]bool, exceptions []string) error {
	t := v.Type()
	exMap := make(map[string]bool)
	for _, ex := range exceptions {
		exMap[strings.ToUpper(ex)] = true
	}

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		fn := strings.ToUpper(t.Field(i).Name)

		if bt != nil && bt[fn] {
			if f.Kind() != reflect.Bool {
				return fmt.Errorf("%s must be boolean", fn)
			}
			continue
		}

		switch f.Kind() {
		case reflect.String:
			if exMap[fn] {
				continue
			}

			if f.String() == "" {
				return fmt.Errorf("%s is required", fn)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if exMap[fn] {
				continue
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if exMap[fn] {
				continue
			}
		case reflect.Float32, reflect.Float64:
			if exMap[fn] {
				continue
			}
		default:
			return fmt.Errorf("unsupported type %s for field %s", f.Kind(), fn)
		}
	}

	return nil
}

// autoMapEnvKeys maps environment variables to the struct fields
func autoMapEnvKeys(v *viper.Viper, val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		f := val.Field(i)
		t := typ.Field(i)

		if f.Kind() == reflect.Struct {
			if err := autoMapEnvKeys(v, f); err != nil {
				return err
			}
			continue
		}

		envVar := t.Tag.Get("map-structure")
		if envVar == "" {
			continue
		}

		switch f.Kind() {
		case reflect.String:
			f.SetString(v.GetString(envVar))
		case reflect.Bool:
			f.SetBool(v.GetBool(envVar))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			f.SetInt(v.GetInt64(envVar))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			f.SetUint(v.GetUint64(envVar))
		case reflect.Float32, reflect.Float64:
			f.SetFloat(v.GetFloat64(envVar))
		default:
			return fmt.Errorf("unsupported type %s", f.Kind())
		}
	}

	return nil
}

// matchPattern matches a pattern against a value
func matchPattern(pattern, value string) bool {
	matched, _ := regexp.MatchString(pattern, value)
	return matched
}
