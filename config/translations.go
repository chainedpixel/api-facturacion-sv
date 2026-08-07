package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

var (
	translateMutex sync.RWMutex
	translations   = make(map[string]map[string]string)
	initialized    = false
	globalLang     = "en"
)

// InitTranslations loads the translation files at startup
func InitTranslations(configPath, lang string) error {
	translateMutex.Lock()
	defer translateMutex.Unlock()

	if initialized {
		return nil
	}

	globalLang = strings.ToLower(lang)
	v := viper.New()

	errorsFile := filepath.Join(configPath, fmt.Sprintf("%s.yaml", globalLang))
	v.SetConfigFile(errorsFile)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return err
	}
	processKeys(v, globalLang)

	initialized = true
	return nil
}

// processKeys recursively processes the keys from the configuration file
func processKeys(v *viper.Viper, lang string) {
	if translations[lang] == nil {
		translations[lang] = make(map[string]string)
	}

	allKeys := v.AllKeys()

	for _, key := range allKeys {
		value := v.Get(key)

		switch val := value.(type) {
		case string:
			translations[lang][key] = val
		case map[string]interface{}:
			for subKey, subVal := range val {
				if strVal, ok := subVal.(string); ok {
					nestedKey := key + "." + subKey
					translations[lang][nestedKey] = strVal
				}
			}
		}
	}
}

func TranslateServiceArgs(code string, params ...interface{}) string {
	code = fmt.Sprintf("service_errors.%s", strings.ToLower(code))
	return Translate(code, params...)
}

// Translate retrieves the translation for a given code and language
func Translate(code string, params ...interface{}) string {
	code = strings.ToLower(code)
	translateMutex.RLock()
	defer translateMutex.RUnlock()

	if !initialized {
		return code
	}

	normalizedLang := normalizeLanguage(globalLang)
	template, found := translations[normalizedLang][code]
	if !found {
		template, found = translations["en"][code]
		if !found {
			return fmt.Sprintf("[%s] %s", code, translations[normalizedLang]["validation_errors.unknownerror"])
		}
	}

	if len(params) > 0 && strings.Contains(template, "%") {
		return fmt.Sprintf(template, params...)
	}

	return template
}

// normalizeLanguage normalizes the language code
func normalizeLanguage(lang string) string {
	simpleLang := strings.ToLower(lang)
	if len(simpleLang) >= 2 {
		simpleLang = simpleLang[:2]
	}

	switch simpleLang {
	case "es":
		return "es"
	default:
		return "en"
	}
}

func ForceReload(configPath string) error {
	translateMutex.Lock()
	initialized = false
	translateMutex.Unlock()

	return InitTranslations(configPath, "en")
}
