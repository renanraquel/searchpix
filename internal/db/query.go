package db

import (
	"regexp"
	"strconv"
)

// QueryForDriver converte placeholders $1, $2 (Postgres) para ? (SQLite)
func QueryForDriver(query string, driver string) string {
	if driver != "sqlite3" {
		return query
	}
	re := regexp.MustCompile(`\$(\d+)`)
	return re.ReplaceAllStringFunc(query, func(m string) string {
		return "?"
	})
}

// Placeholders retorna n placeholders no formato do driver ($1,$2 para postgres, ?,? para sqlite)
func Placeholders(driver string, n int) string {
	if driver == "sqlite3" {
		s := ""
		for i := 0; i < n; i++ {
			if i > 0 {
				s += ","
			}
			s += "?"
		}
		return s
	}
	s := ""
	for i := 1; i <= n; i++ {
		if i > 1 {
			s += ","
		}
		s += "$" + strconv.Itoa(i)
	}
	return s
}
