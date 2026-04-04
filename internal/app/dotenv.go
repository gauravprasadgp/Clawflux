package app

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

// LoadDotEnv loads simple KEY=VALUE pairs from path into the process environment.
// Existing environment variables win over values in the file.
func LoadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s:%d: expected KEY=VALUE", path, lineNo)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("%s:%d: empty key", path, lineNo)
		}
		if os.Getenv(key) != "" {
			continue
		}

		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}

	return scanner.Err()
}
