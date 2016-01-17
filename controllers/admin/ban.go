// Copyright (C) 2015  TF2Stadium
// Use of this source code is governed by the GPLv3
// that can be found in the COPYING file.

package admin

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/TF2Stadium/Helen/config"
	"github.com/TF2Stadium/Helen/models"
)

var timeRe = regexp.MustCompile(`(\d+[a-z])`)
var banString = map[models.PlayerBanType]string{
	models.PlayerBanJoin:   "joining lobbies",
	models.PlayerBanCreate: "creating lobbies",
	models.PlayerBanChat:   "chatting",
	models.PlayerBanFull:   "the website",
}

//(y)ear (m)onth (w)eek (d)ay (h)our
func parseTime(str string) (*time.Time, error) {
	var year, month, week, day, hour int

	if !timeRe.MatchString(str) {
		return nil, errors.New("Invalid time duration")
	}

	matches := timeRe.FindStringSubmatch(str)
	for _, match := range matches {
		suffix := match[len(match)-1]
		prefix := match[:len(match)-1]
		num, err := strconv.Atoi(prefix)
		if err != nil {
			return nil, err
		}

		switch suffix {
		case 'y':
			year = num
		case 'm':
			month = num
		case 'w':
			week = num
		case 'd':
			day = num
		case 'h':
			hour = num
		}
	}

	t := time.Now().AddDate(year, month, week*7+day).Add(time.Hour * time.Duration(hour))
	return &t, nil
}

func banPlayer(w http.ResponseWriter, r *http.Request, banType models.PlayerBanType) error {
	values := r.URL.Query()
	confirm := values.Get("confirm")
	steamid := values.Get("steamid")
	reason := values.Get("reason")

	player, tperr := models.GetPlayerBySteamID(steamid)
	if tperr != nil {
		return tperr
	}

	switch confirm {
	case "yes":
		until, err := parseTime(values.Get("until"))
		if err != nil {
			return err
		}

		player.BanUntil(*until, banType, reason)
	default:
		query := r.URL.Query()
		query.Set("confirm", "yes")
		r.URL.RawQuery = query.Encode()
		title := fmt.Sprintf("Ban %s (%s) from %s?", player.Name, player.SteamID, banString[banType])
		confirmReq(w, r.URL.String(), config.Constants.Domain+"/admin/ban", title)
	}

	return nil
}

func BanJoin(w http.ResponseWriter, r *http.Request) {
	err := banPlayer(w, r, models.PlayerBanJoin)
	if err != nil {
		http.Error(w, err.Error(), 400)
	}
}

func BanChat(w http.ResponseWriter, r *http.Request) {
	err := banPlayer(w, r, models.PlayerBanChat)
	if err != nil {
		http.Error(w, err.Error(), 400)
	}
}

func BanCreate(w http.ResponseWriter, r *http.Request) {
	err := banPlayer(w, r, models.PlayerBanCreate)
	if err != nil {
		http.Error(w, err.Error(), 400)
	}
}

func BanFull(w http.ResponseWriter, r *http.Request) {
	err := banPlayer(w, r, models.PlayerBanFull)
	if err != nil {
		http.Error(w, err.Error(), 400)
	}
}
