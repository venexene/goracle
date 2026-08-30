package database

import (
	"errors"
	"strings"
)

func UsersQuery(sort, direction string) (string, error) {
	columns := map[string]string{"created": "created_at", "name": "display_name"}
	directions := map[string]string{"asc": "ASC", "desc": "DESC"}
	column, ok := columns[sort]
	if !ok {
		return "", errors.New("неподдерживаемое поле сортировки")
	}
	order, ok := directions[direction]
	if !ok {
		return "", errors.New("неподдерживаемое направление сортировки")
	}
	return "SELECT id, display_name FROM users ORDER BY " + column + " " + order, nil
}

func EscapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}

func Retry(attempts int, transaction func() error, retryable func(error) bool) error {
	var err error
	for range attempts {
		if err = transaction(); err == nil || !retryable(err) {
			return err
		}
	}
	return err
}
