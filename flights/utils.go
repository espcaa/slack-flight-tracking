package flights

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func GetAirlinesDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:airlines.db?cache=shared&mode=ro")
	if err != nil {
		return nil, err
	}
	return db, nil
}

func GetNameFromICAO(db *sql.DB, icao string) (string, error) {
	var name string
	err := db.QueryRow("SELECT name FROM airlines WHERE icao = ?", icao).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}

func GetICAOFromIATA(db *sql.DB, iata string) (string, error) {
	var icao string
	err := db.QueryRow("SELECT icao FROM airlines WHERE iata = ?", iata).Scan(&icao)
	if err != nil {
		return "", err
	}
	return icao, nil
}
