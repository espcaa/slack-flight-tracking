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

func ListFlightsByUser(userID string, config Config) ([]Flight, error) {
	rows, err := config.UserDB.Query("SELECT id, flight_number, slack_channel, slack_user_id, departure FROM flights WHERE slack_user_id=$1", userID)
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

func UntrackFlight(id string, config Config) error {
	_, err := config.UserDB.Exec("DELETE FROM flights WHERE id=$1", id)
	return err
}

func FindFlight(flightNumber, slackChannel, slackUserID string, config Config) (*Flight, error) {
	row := config.UserDB.QueryRow("SELECT id, flight_number, slack_channel, slack_user_id, departure FROM flights WHERE flight_number=$1 AND slack_channel=$2 AND slack_user_id=$3", flightNumber, slackChannel, slackUserID)

	var f Flight
	err := row.Scan(&f.ID, &f.FlightNumber, &f.SlackChannel, &f.SlackUserID, &f.Departure)
	if err != nil {
		return nil, err
	}
	return &f, nil
}
