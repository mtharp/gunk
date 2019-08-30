package model

import "time"

type ChannelInfo struct {
	Name    string `json:"name"`
	Live    bool   `json:"live"`
	Last    int64  `json:"last"`
	Thumb   string `json:"thumb"`
	LiveURL string `json:"live_url"`
	Viewers int    `json:"viewers"`
}

func ListChannelInfo() (ret []*ChannelInfo, err error) {
	rows, err := db.Query("SELECT name, updated FROM thumbs ORDER BY greatest(now() - updated, '1 minute'::interval) ASC, 1 ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		info := new(ChannelInfo)
		var last time.Time
		if err := rows.Scan(&info.Name, &last); err != nil {
			return nil, err
		}
		info.Last = last.UnixNano() / 1000000
		ret = append(ret, info)
	}
	err = rows.Err()
	return
}
