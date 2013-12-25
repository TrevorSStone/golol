package rtmpsclient

import (
	"encoding/hex"
	"fmt"
	"github.com/TrevorSStone/goamf"
	"runtime"
	"testing"
	"time"
)

var (
	username      = "yourusername"
	password      = "yourpassword"
	leagueversion = "currentleagueversion"
)

func TestConnection(t *testing.T) {

	connection := New("prod.na1.lol.riotgames.com:2099", "", "app:/mod_ser.dat", "null")
	err := connection.Connect()
	fmt.Println("Test Connected")
	if err != nil {
		fmt.Println(err.Error())
		t.Error("lol")
	}

}

func TestLogin(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	connection := New("prod.na1.lol.riotgames.com:2099", "", "app:/mod_ser.dat", "")
	err := connection.Connect()
	if err != nil {
		fmt.Println(err.Error())
		t.Error("lol")
	}
	err = connection.Login(username, password, leagueversion)
	if err != nil {

		t.Error(err.Error())
	}
}

func TestGetIPAddress(t *testing.T) {
	ip, err := getIPAddress()
	fmt.Println(ip.String())
	if err != nil {
		t.Error(err.Error())
	}
}

func TestGetAuthToken(t *testing.T) {
	token, err := getAuthToken(username, password, LeagueServerInfo["NA"])
	if err != nil {
		t.Error(err.Error())
	} else {
		fmt.Println(token)
	}
}

func TestDecode(t *testing.T) {
	data, _ := hex.DecodeString("000200075F726573756C7400400000000000000005110A070744534BA1030A3349636F6D2E72696F7467616D65732E706C6174666F726D2E6C6F67696E2E53657373696F6E0B746F6B656E1170617373776F72641D6163636F756E7453756D6D617279064933396262363837342D366136642D343838332D623562642D3961643664633836323432660639746869736163636F756E746973666F7274657374696E676F6E6C79310A81335B636F6D2E72696F7467616D65732E706C6174666F726D2E6163636F756E742E4163636F756E7453756D6D6172791567726F7570436F756E7411757365726E616D65136163636F756E7449642973756D6D6F6E6572496E7465726E616C4E616D65176461746156657273696F6E0B61646D696E1B686173426574614163636573731973756D6D6F6E65724E616D6517706172746E65724D6F6465256E6565647350617373776F7264526573657415667574757265446174610400060F63686174626F740541813D873800000001040002030102020105427425E7FA8370000C218B5A8C6587E7989639BB67A877C66AA30C218B5A9AE471B68AFF62389DEB8E57CF7801064935636535363833612D643435372D346233362D366438392D38376237613563653130613100")

	DecodeMessage(data)
}

func TestMessageManager(t *testing.T) {
	lookupRequestChan := make(chan MessageLookup)
	storeRequestChan := make(chan amf.Object)
	go messageManager(lookupRequestChan, storeRequestChan)
	go func() {
		asynctest := NewMessageLookup(2)
		lookupRequestChan <- asynctest
		fmt.Println("agsggas")
		lol := <-asynctest.ReturnChan
		fmt.Println("gorout1")
		fmt.Println(lol)
	}()
	go func() {
		asynctest := NewMessageLookup(2)
		lookupRequestChan <- asynctest
		lol := <-asynctest.ReturnChan
		fmt.Println("gorout2")
		fmt.Println(lol)
	}()
	go func() {
		asynctest := NewMessageLookup(2)
		lookupRequestChan <- asynctest
		lol := <-asynctest.ReturnChan
		fmt.Println("gorout3")
		fmt.Println(lol)
	}()
	go func() {
		asynctest := NewMessageLookup(2)
		lookupRequestChan <- asynctest
		lol := <-asynctest.ReturnChan
		fmt.Println("gorout4")
		fmt.Println(lol)
	}()
	time.Sleep(2 * time.Second)
	fmt.Println("after sleep")
	test := amf.Object{"hello": "hi", "invokeId": 2}
	storeRequestChan <- test

	test = amf.Object{"hello": "hi", "invokeId": 2}
	storeRequestChan <- test
	fmt.Println("aassfsfasfasf")
	test2 := NewMessageLookup(2)
	lookupRequestChan <- test2
	lol := <-test2.ReturnChan
	fmt.Println(lol)
	storeRequestChan <- test
	time.Sleep(2 * time.Second)
	fmt.Println("after sleep2")
	test3 := NewMessageLookup(2)
	lookupRequestChan <- test3
	lol2 := <-test3.ReturnChan
	fmt.Println(lol2)
}

func TestInvokeIDGenerator(t *testing.T) {
	idChannel := make(chan int)
	go InvokeIdGenerator(idChannel)
}

func TestGetLogin1Data(t *testing.T) {
	test := amf.Object{
		"result": "_result",
		"data":   amf.Object{},
	}
	_, _, err := getLogin1Data(test)
	fmt.Println(err)
}

func TestTesting(t *testing.T) {

	fmt.Println(time.Now().Format("002 Jan 2 2006 15:04:05 GMTZ"))
}
