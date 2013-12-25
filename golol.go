/*
Package golol is a library that provides a means of connecting and recieving data from Riot's League of Legend Servers
*/
package golol

import (
	"encoding/hex"
	"errors"
	"fmt"
	"golol/rtmpsclient"
	"strconv"
	"time"
)

//Might need to worry about ClientVersion being different on different Server
var (
	ClientVersion = "3.15.13_12_13_16_07"
	port          = 2099
)

type LeagueConnection struct {
	connection rtmpsclient.RTMPSClient
}

type LoginInfo struct {
	ServerName string
	Username   string
	Password   string
}

type LeaguePool chan LeagueConnection

func NewPool(loginInfo ...LoginInfo) LeaguePool {
	pool := make(LeaguePool, len(loginInfo))
	finishChan := make(chan bool)
	for _, v := range loginInfo {
		go newPoolConnection(v, pool, finishChan)
	}
	for i := 0; i < len(loginInfo); i++ {
		<-finishChan
	}
	fmt.Println("finished logging in")
	return pool
}

func newPoolConnection(loginInfo LoginInfo, pool LeaguePool, finishChan chan<- bool) {
	connection, err := New(loginInfo.ServerName, loginInfo.Username, loginInfo.Password)
	if err != nil {
		fmt.Printf("%s error logging in: %s", loginInfo.Username, err.Error())

	} else {
		pool <- connection
	}
	finishChan <- true
}

func New(serverName, username, password string) (LeagueConnection, error) {
	serverinfo, ok := rtmpsclient.LeagueServerInfo[serverName]
	if !ok {
		return LeagueConnection{}, errors.New("Not A Valid Server Name")
	}
	serverString := fmt.Sprintf("%s:%d", serverinfo.Server.Host, port)
	connection := rtmpsclient.New(serverString, "", "app:/mod_ser.dat", "")
	err := connection.Connect()
	if err != nil {
		return LeagueConnection{}, err
	}
	err = connection.Login(username, password, ClientVersion)
	if err != nil {
		return LeagueConnection{}, err
	}
	return LeagueConnection{
		connection: connection,
	}, nil
}

func (pool LeaguePool) GetNextConnection() (client LeagueConnection, err error) {
	select {
	case client = <-pool:
		pool <- client
	case <-time.After(10 * time.Second):
		err = errors.New("Could not find available connection in 10s")
	}
	return
}

func (pool LeaguePool) GetSummonerByName(summonerName string) (summoner Summoner, err error) {
	client, err := pool.GetNextConnection()
	if err != nil {
		return summoner, err
	}
	return client.GetSummonerByName(summonerName)

}

func (client LeagueConnection) GetSummonerByName(summonerName string) (summoner Summoner, err error) {
	response, err := client.connection.BlockingRequest("summonerService", "getSummonerByName", summonerName, 10)
	if err != nil {
		return summoner, err
	}
	return unmarshalSummoner(response)
}

func (pool LeaguePool) GetSummonerRunePages(summonerID int) (runePages []RunePage, err error) {
	client, err := pool.GetNextConnection()
	if err != nil {
		return runePages, err
	}
	return client.GetSummonerRunePages(summonerID)

}

func (client LeagueConnection) GetSummonerRunePages(summonerID int) (runePages []RunePage, err error) {
	response, err := client.connection.BlockingRequest("summonerService", "getAllPublicSummonerDataByAccount", summonerID, 10)
	if err != nil {
		return runePages, err
	}
	return unmarshalRunePages(response)
}

func (pool LeaguePool) GetSummonerRunePages2(summonerID int) {
	client, err := pool.GetNextConnection()
	if err != nil {
		return
	}
	client.GetSummonerRunePages2(summonerID)

}

func (client LeagueConnection) GetSummonerRunePages2(summonerID int) {
	response, err := client.connection.BlockingRequest("masteryBookService", "getMasteryBook", summonerID, 10)
	if err != nil {
		return
	}
	fmt.Println(response)
}

// id = client.invoke("summonerService", "getSummonerByName", new Object[] { "ManticoreX" });

func ChangeClientVersion(version string) {
	ClientVersion = version
}

func ChangePort(newPort int) {
	port = newPort
}

func GetLeagueID(summonerName string) (int, error) {

	encode := hex.EncodeToString([]byte(summonerName))
	value, _ := strconv.ParseUint(encode, 16, 64)
	value = value % 10000
	return int(value), nil
}

func GetRunePages(leagueID int) bool {
	time.Sleep(time.Duration(2) * time.Minute)
	return true
}
