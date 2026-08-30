package database

import "errors"

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
