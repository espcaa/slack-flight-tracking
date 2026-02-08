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
