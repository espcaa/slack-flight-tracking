package shared

func RegisterTrackedFlight(flight Flight, config Config) error {
	// add the flight to the database
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
	ID           string
	FlightNumber string
	SlackChannel string
	SlackUserID  string
}

func GetFlightState(flightID string, config Config) (*FlightState, error) {
	row := config.UserDB.QueryRow(`SELECT flight_id, status, origin_gate, dest_gate,
		dep_scheduled, dep_estimated, dep_actual, arr_scheduled, arr_estimated, arr_actual
		FROM flight_state WHERE flight_id = ?`, flightID)

	var s FlightState
	err := row.Scan(&s.FlightID, &s.Status, &s.OriginGate, &s.DestGate,
		&s.DepScheduled, &s.DepEstimated, &s.DepActual, &s.ArrScheduled, &s.ArrEstimated, &s.ArrActual)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func SaveFlightState(state FlightState, config Config) error {
	_, err := config.UserDB.Exec(`INSERT INTO flight_state (flight_id, status, origin_gate, dest_gate,
		dep_scheduled, dep_estimated, dep_actual, arr_scheduled, arr_estimated, arr_actual)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(flight_id) DO UPDATE SET
		status=excluded.status, origin_gate=excluded.origin_gate,
		dest_gate=excluded.dest_gate,
		dep_scheduled=excluded.dep_scheduled, dep_estimated=excluded.dep_estimated, dep_actual=excluded.dep_actual,
		arr_scheduled=excluded.arr_scheduled, arr_estimated=excluded.arr_estimated, arr_actual=excluded.arr_actual`,
		state.FlightID, state.Status, state.OriginGate, state.DestGate,
		state.DepScheduled, state.DepEstimated, state.DepActual, state.ArrScheduled, state.ArrEstimated, state.ArrActual)
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
