package config

// envConfig is a struct that holds the configuration from the .env file
type envConfig struct {
	Server   server
	Database database
	Redis    redis
	Log      log
	Signer   signer
	MHPaths  mhPaths
	SMTP     smtp
}

// server is a struct that holds the server configuration
type server struct {
	Port                  string `map-structure:"SERVER_PORT"`
	MaxBatchSize          int    `map-structure:"MH_MAX_BATCH_SIZE"`
	JWTSecret             string `map-structure:"JWT_SECRET"`
	AmbientCode           string `map-structure:"MH_AMBIENT_CODE"`
	Debug                 bool   `map-structure:"DEBUG"`
	RunMigration          bool   `map-structure:"RUN_MIGRATION"`
	AdminEmail            string `map-structure:"ADMIN_EMAIL"`
	ForceContingency      bool   `map-structure:"FORCE_CONTINGENCY"`
	AppLang               string `map-structure:"APP_LANG"`
	AppLangPath           string `map-structure:"APP_LANG_PATH"`
	APIVersion            string `map-structure:"API_VERSION"`
	NotifyCooldownMinutes int    `map-structure:"NOTIFY_COOLDOWN_MINUTES"`
}

// smtp is a struct that holds the SMTP configuration
type smtp struct {
	Host     string `map-structure:"SMTP_HOST"`
	Port     string `map-structure:"SMTP_PORT"`
	Username string `map-structure:"SMTP_USERNAME"`
	Password string `map-structure:"SMTP_PASSWORD"`
	From     string `map-structure:"SMTP_FROM"`
	TLS      bool   `map-structure:"SMTP_TLS"`
}

// database is a struct that holds the database configuration
type database struct {
	Host     string `map-structure:"DB_HOST"`
	Port     string `map-structure:"DB_PORT"`
	Name     string `map-structure:"DB_DATABASE"`
	User     string `map-structure:"DB_USERNAME"`
	Password string `map-structure:"DB_PASSWORD"`
	Charset  string `map-structure:"DB_CHARSET"`
	Driver   string `map-structure:"DB_DRIVER"`
}

// redis is a struct that holds the Redis configuration
type redis struct {
	Host     string `map-structure:"REDIS_HOST"`
	Port     string `map-structure:"REDIS_PORT"`
	Password string `map-structure:"REDIS_PASSWORD"`
}

// log is a struct that holds the logging configuration
type log struct {
	Level       string `map-structure:"LOG_LEVEL"`
	Path        string `map-structure:"LOG_PATH"`
	FileLogging bool   `map-structure:"LOG_FILE_LOGGING"`
}

// signer is a struct that holds the signer configuration
type signer struct {
	Path   string `map-structure:"SIGNER_PATH"`
	Health string `map-structure:"SIGNER_HEALTH"`
}

// mhPaths is a struct that holds the MH service endpoint paths
type mhPaths struct {
	AuthURL                 string `map-structure:"MH_AUTH_URL"`
	ReceptionURL            string `map-structure:"MH_RECEPTION_URL"`
	LoteReceptionURL        string `map-structure:"MH_LOTE_RECEPTION_URL"`
	ReceptionConsultURL     string `map-structure:"MH_RECEPTION_CONSULT_URL"`
	LoteReceptionConsultURL string `map-structure:"MH_RECEPTION_CONSULT_LOTE_URL"`
	ContingencyURL          string `map-structure:"MH_CONTINGENCY_URL"`
	NullifyURL              string `map-structure:"MH_NULLIFY_URL"`
}
