package rtmpsclient

import (
	"net/url"
)

type ServerInfo struct {
	Region     string
	Platform   string
	Name       string
	Server     url.URL
	LoginQueue url.URL
	IsGarena   bool
}

var LeagueServerInfo = map[string]ServerInfo{
	"NA": ServerInfo{
		Region:   "NA",
		Platform: "NA1",
		Name:     "North America",
		Server: url.URL{
			Host: "prod.na1.lol.riotgames.com",
		},
		LoginQueue: url.URL{
			Scheme: "https",
			Host:   "lq.na1.lol.riotgames.com",
		},
	},
	"EUW": ServerInfo{
		Region:   "EUW",
		Platform: "EUW1",
		Name:     "Europe West",
		Server: url.URL{
			Host: "prod.eu.lol.riotgames.com",
		},
		LoginQueue: url.URL{
			Scheme: "https",
			Host:   "lq.eu.lol.riotgames.com",
		},
	},
	"EUNE": ServerInfo{
		Region:   "EUNE",
		Platform: "EUN1",
		Name:     "Europe Nordic & East",
		Server: url.URL{
			Host: "prod.eun1.lol.riotgames.com",
		},
		LoginQueue: url.URL{
			Scheme: "https",
			Host:   "lq.eun1.lol.riotgames.com",
		},
	},
}
