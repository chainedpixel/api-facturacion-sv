package utils

import (
	"fmt"
	"strings"

	"github.com/chainedpixel/ordo-factus/config"
)

func TranslateMessage(key, code string, params ...interface{}) string {
	code = strings.ToLower(fmt.Sprintf("%s.%s", key, code))
	return config.Translate(code, params...)
}

func TranslateHealthUp(key string) string {
	return TranslateMessage("health.up", key)
}

func TranslateHealthDown(key string) string {
	return TranslateMessage("health.down", key)
}

func TranslateHealthError(key string, params ...interface{}) string {
	return TranslateMessage("health.error", key, params...)
}

func TranslateHealthNotConfigured(key string) string {
	return TranslateMessage("health.notconfigured", key)
}
