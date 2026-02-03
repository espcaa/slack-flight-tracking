package flights

import (
	"database/sql"
	"errors"
	"regexp"

	_ "modernc.org/sqlite"
)

// Regex patterns for airline codes
var (
	IcaoPattern      = regexp.MustCompile(`^[A-Z]{3}$`)                // ICAO airline code (3 uppercase letters)
	IataPattern      = regexp.MustCompile(`^[A-Z]{2}$`)                // IATA airline code (2 uppercase letters)
	FlightNumPattern = regexp.MustCompile(`^[A-Z]{2,3}\d{1,4}[A-Z]?$`) // Flight number pattern (e.g., "AA100", "DLH400A"
)

var db *sql.DB

func init() {
	// init the db connection
	var err error
	db, err = GetAirlinesDB()
	if err != nil {
		panic("failed to connect to airlines database: " + err.Error())
	}
}

// GetAirlinesDB opens a read-only connection to the airlines database
func GetAirlinesDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:data/airlines.db?cache=shared&mode=ro")
	if err != nil {
		return nil, err
	}
	return db, nil
}

// GetAirlineNameFromICAO looks up the airline name given an ICAO code
func GetAirlineNameFromICAO(db *sql.DB, icao string) (string, error) {
	if !IcaoPattern.MatchString(icao) {
		return "", errors.New("invalid ICAO code format")
	}

	var name string
	err := db.QueryRow("SELECT name FROM airlines WHERE icao = ?", icao).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}

// GetAirlineICAOFromIATA looks up the ICAO code given an IATA code
func GetAirlineICAOFromIATA(db *sql.DB, iata string) (string, error) {
	if !IataPattern.MatchString(iata) {
		return "", errors.New("invalid IATA code format")
	}

	var icao string
	err := db.QueryRow("SELECT icao FROM airlines WHERE iata = ?", iata).Scan(&icao)
	if err != nil {
		return "", err
	}
	return icao, nil
}

// AirlineCodeToICAO converts either IATA or ICAO airline code to ICAO
// e.g., "AF" → "AFR", or returns the ICAO itself if already ICAO
func AirlineCodeToICAO(db *sql.DB, code string) (string, error) {
	// Already ICAO?
	if IcaoPattern.MatchString(code) {
		return code, nil
	}

	// Possibly IATA
	if IataPattern.MatchString(code) {
		icao, err := GetAirlineICAOFromIATA(db, code)
		if err != nil {
			return "", err
		}
		return icao, nil
	}

	return "", errors.New("invalid airline code format")
}

// ExpandFlightNumber ensures a flight number uses the ICAO airline code
// e.g., "AF102" → "AFR102"
func ExpandFlightNumber(flight string) (string, error) {

	// Regex to split letters vs digits
	var flightParts = regexp.MustCompile(`^([A-Z]{2,3})(\d{1,4}[A-Z]?)$`)
	matches := flightParts.FindStringSubmatch(flight)
	if matches == nil {
		return "", errors.New("invalid flight number format")
	}

	code := matches[1] // airline code (IATA or ICAO)
	num := matches[2]  // numeric part

	icao, err := AirlineCodeToICAO(db, code)
	if err != nil {
		return "", err
	}

	return icao + num, nil
}
