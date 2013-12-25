/*
Package golol is a library that provides a means of connecting and recieving data from Riot's League of Legend Servers
*/
package golol

import (
	"fmt"
	"testing"
)

var (
	username1 = "yourusername"
	password1 = "yourpassword"
	username2 = "yourusername2"
	password2 = "yourpassword2"
)

func TestGetSummonerByName(t *testing.T) {
	leagueconn, err := New("NA", username1, password1)
	if err != nil {
		t.Fatal(err.Error())
	}
	leagueconn.GetSummonerByName("Jabe")
	leagueconn.GetSummonerByName("ManticoreX")

}

func TestLoginPool(t *testing.T) {
	chatbot := LoginInfo{
		Username:   username1,
		ServerName: "NA",
		Password:   password1,
	}
	chin := LoginInfo{
		Username:   username2,
		ServerName: "NA",
		Password:   password2,
	}
	pool := NewPool(chatbot, chin)
	//for i := 0; i < 100; i++ {
	fmt.Println(pool.GetSummonerByName("Jabe"))
	fmt.Println(pool.GetSummonerByName("ManticoreX"))
	fmt.Println(pool.GetSummonerByName("ChinchillaKing"))
	pages, err := pool.GetSummonerRunePages(28976)
	if err != nil {
		t.Error(err)
	}
	for _, v := range pages {
		fmt.Println(v.Name)
	}
	pages, err = pool.GetSummonerRunePages(36691274)
	if err != nil {
		t.Error(err)
	}
	for _, v := range pages {
		fmt.Println(v.Name)
	}
	//}
}
