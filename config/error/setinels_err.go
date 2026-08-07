package error

import "errors"

var (
	ErrFailedToConnectDb         = errors.New("failed to connect to database")
	ErrFailedToCloseDbConnection = errors.New("failed to close database connection")
	ErrFailedToGetDBInstance     = errors.New("failed to get database instance")
	ErrEnvFileNotFound           = errors.New("env file not found")
	ErrFailedToLoadEnv           = errors.New("failed to load env file")
	ErrUnrecognizedDriver        = errors.New("unrecognized database driver")
)
