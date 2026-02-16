package shared

import (
	"fmt"
	"reflect"
	"strings"
)

func structColumns(ptr any) (columns []string, scanDest []any) {
	v := reflect.ValueOf(ptr).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		col := t.Field(i).Tag.Get("db")
		if col == "" {
			continue
		}
		columns = append(columns, col)
		scanDest = append(scanDest, v.Field(i).Addr().Interface())
	}
	return
}

func placeholders(n int) string {
	return strings.Repeat("?, ", n-1) + "?"
}

func upsertSet(columns []string, skip string) string {
	parts := make([]string, 0, len(columns))
	for _, c := range columns {
		if c == skip {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=excluded.%s", c, c))
	}
	return strings.Join(parts, ", ")
}

func RegisterTrackedFlight(flight Flight, config Config) error {
	_, err := config.UserDB.Exec("INSERT INTO flights (id, flight_number, slack_channel, slack_user_id, departure) VALUES ($1, $2, $3, $4, $5)", flight.ID, flight.FlightNumber, flight.SlackChannel, flight.SlackUserID, flight.Departure)
	return err
}

func GetFlight(id string, config Config) (*Flight, error) {
	row := config.UserDB.QueryRow("SELECT id, flight_number, slack_channel, slack_user_id, departure FROM flights WHERE id=$1", id)

	var f Flight
	err := row.Scan(&f.ID, &f.FlightNumber, &f.SlackChannel, &f.SlackUserID, &f.Departure)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func UntrackFlight(id string, config Config) error {
	_, err := config.UserDB.Exec("DELETE FROM flights WHERE id=$1", id)
	return err
}

type FlightFilter struct {
	ID             string
	FlightNumber   string
	SlackChannel   string
	SlackUserID    string
	DepartureAfter int64
}

func GetFlightState(flightID string, config Config) (*FlightState, error) {
	var s FlightState
	cols, dest := structColumns(&s)
	query := fmt.Sprintf("SELECT %s FROM flight_state WHERE flight_id = ?", strings.Join(cols, ", "))

	err := config.UserDB.QueryRow(query, flightID).Scan(dest...)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func SaveFlightState(state FlightState, config Config) error {
	cols, _ := structColumns(&state)
	vals := make([]any, 0, len(cols))
	v := reflect.ValueOf(state)
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Tag.Get("db") != "" {
			vals = append(vals, v.Field(i).Interface())
		}
	}

	query := fmt.Sprintf(`INSERT INTO flight_state (%s) VALUES (%s)
		ON CONFLICT(flight_id) DO UPDATE SET %s`,
		strings.Join(cols, ", "),
		placeholders(len(cols)),
		upsertSet(cols, "flight_id"))

	_, err := config.UserDB.Exec(query, vals...)
	return err
}

func AlertAlreadySent(flightID, alertType string, config Config) bool {
	row := config.UserDB.QueryRow("SELECT 1 FROM alerts_sent WHERE flight_id = ? AND alert_type = ?", flightID, alertType)
	var exists int
	return row.Scan(&exists) == nil
}

func MarkAlertSent(flightID, alertType string, config Config) error {
	_, err := config.UserDB.Exec("INSERT OR IGNORE INTO alerts_sent (flight_id, alert_type) VALUES (?, ?)", flightID, alertType)
	return err
}

func GetFlights(filter FlightFilter, config Config) ([]Flight, error) {
	query := "SELECT id, flight_number, slack_channel, slack_user_id, departure FROM flights WHERE 1=1"
	args := []any{}

	if filter.ID != "" {
		query += " AND id = ?"
		args = append(args, filter.ID)
	}
	if filter.FlightNumber != "" {
		query += " AND flight_number = ?"
		args = append(args, filter.FlightNumber)
	}
	if filter.SlackChannel != "" {
		query += " AND slack_channel = ?"
		args = append(args, filter.SlackChannel)
	}
	if filter.SlackUserID != "" {
		query += " AND slack_user_id = ?"
		args = append(args, filter.SlackUserID)
	}
	if filter.DepartureAfter != 0 {
		query += " AND departure > ?"
		args = append(args, filter.DepartureAfter)
	}

	rows, err := config.UserDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flights []Flight
	for rows.Next() {
		var f Flight
		err := rows.Scan(&f.ID, &f.FlightNumber, &f.SlackChannel, &f.SlackUserID, &f.Departure)
		if err != nil {
			return nil, err
		}
		flights = append(flights, f)
	}
	return flights, nil
}
