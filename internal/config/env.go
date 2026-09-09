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

// loadEnvFile читает key=value файл и выставляет переменные окружения,
// не затирая уже существующие. Отсутствие файла не является ошибкой.
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

		// Пропускаем пустые строки и комментарии.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Поддержка синтаксиса "export KEY=VALUE".
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

		// Переменные окружения имеют приоритет над файлом.
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

// trimQuotes снимает обрамляющие одинарные или двойные кавычки.
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

// lookupString возвращает строковое значение переменной окружения
// либо значение по умолчанию.
func lookupString(key, def string) string {
	if raw, ok := os.LookupEnv(key); ok {
		if trimmed := strings.TrimSpace(raw); trimmed != "" {
			return trimmed
		}
	}
	return def
}

// lookupStringFirst возвращает значение первого найденного ключа из списка.
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

// lookupBool возвращает булево значение переменной окружения.
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

// lookupDuration возвращает длительность (например "10s", "2m").
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

// lookupIDList парсит список Telegram ID, разделённых запятыми или пробелами.
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
