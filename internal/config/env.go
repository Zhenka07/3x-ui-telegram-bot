package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// loadEnvFile reads a key=value file and sets environment variables without overwriting existing ones.
func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("открытие env-файла %q: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")

		key, value, found := strings.Cut(line, "=")
		if !found {
			return fmt.Errorf("некорректная строка %d в %q: ожидался формат KEY=VALUE", lineNo, path)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("пустой ключ в строке %d файла %q", lineNo, path)
		}

		value = strings.TrimSpace(value)
		value = trimQuotes(value)

		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("установка переменной %q: %w", key, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("чтение env-файла %q: %w", path, err)
	}
	return nil
}

// trimQuotes removes surrounding single or double quotes from a string.
func trimQuotes(value string) string {
	if len(value) < 2 {
		return value
	}
	first, last := value[0], value[len(value)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}

// lookupString returns the environment variable value for key or a default value if not set.
func lookupString(key, def string) string {
	if raw, ok := os.LookupEnv(key); ok {
		if trimmed := strings.TrimSpace(raw); trimmed != "" {
			return trimmed
		}
	}
	return def
}

// lookupStringFirst returns the value of the first matching environment variable key or a default value.
func lookupStringFirst(keys []string, def string) string {
	for _, key := range keys {
		if raw, ok := os.LookupEnv(key); ok {
			if trimmed := strings.TrimSpace(raw); trimmed != "" {
				return trimmed
			}
		}
	}
	return def
}

// lookupBool parses a boolean environment variable or returns the default value.
func lookupBool(key string, def bool) (bool, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return def, nil
	}
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return false, fmt.Errorf("переменная %s: ожидалось true/false, получено %q: %w", key, raw, err)
	}
	return value, nil
}

// lookupDuration parses a duration environment variable or returns the default value.
func lookupDuration(key string, def time.Duration) (time.Duration, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return def, nil
	}
	value, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("переменная %s: некорректная длительность %q: %w", key, raw, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("переменная %s: длительность должна быть положительной, получено %q", key, raw)
	}
	return value, nil
}

// lookupIDList parses a list of Telegram IDs separated by commas, spaces, or semicolons.
func lookupIDList(key string) ([]int64, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';' || r == '\t'
	})

	ids := make([]int64, 0, len(fields))
	for _, field := range fields {
		id, err := strconv.ParseInt(strings.TrimSpace(field), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("переменная %s: некорректный Telegram ID %q: %w", key, field, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
