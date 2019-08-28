package model

func GetThumb(channelName string) (d []byte, err error) {
	row := db.QueryRow("SELECT thumb FROM thumbs WHERE name = $1", channelName)
	err = row.Scan(&d)
	return
}

func PutThumb(channelName string, d []byte) error {
	_, err := db.Exec("INSERT INTO thumbs (name, thumb) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET thumb = EXCLUDED.thumb, updated = now()", channelName, d)
	return err
}
