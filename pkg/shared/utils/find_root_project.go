package utils

import (
	"os"
	"path/filepath"
)

// FindProjectRoot busca hacia arriba hasta encontrar el directorio raíz del proyecto y regresa su ruta
func FindProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	currentDir, _ := os.Getwd()
	return currentDir
}
