package rtmpsclient

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TrevorSStone/goamf"
	"github.com/nu7hatch/gouuid"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type RTMPSClient struct {
	passphrase         string
	addr               string
	app                string
	swfURL             string
	pageURL            string
	ipAddress          net.IP
	dSID               string
	conn               net.Conn
	writeChan          chan []byte
	lookupRequestChan  chan MessageLookup
	storeRequestChan   chan amf.Object
	idGeneratorChannel chan int
	heartbeatStop      chan bool
	serverinfo         ServerInfo
}

type connectionResponse struct {
	Success bool
	ID      string
	Err     error
}

var startTime time.Time

func init() {
	startTime = time.Now()
}

func New(addr string, app string, swfURL string, pageURL string) RTMPSClient {
	writeChan := make(chan []byte)
	lookupRequestChan := make(chan MessageLookup)
	storeRequestChan := make(chan amf.Object)
	idGeneratorChannel := make(chan int)
	heartbeatStop := make(chan bool)
	return RTMPSClient{
		addr:               addr,
		app:                app,
		swfURL:             swfURL,
		pageURL:            pageURL,
		serverinfo:         LeagueServerInfo["NA"],
		writeChan:          writeChan,
		lookupRequestChan:  lookupRequestChan,
		storeRequestChan:   storeRequestChan,
		idGeneratorChannel: idGeneratorChannel,
		heartbeatStop:      heartbeatStop,
	}
}

//
// If error panic
func CheckError(err error, name string) {
	if err != nil {
		panic(errors.New(fmt.Sprintf("%s: %s", name, err.Error())))
	}
}

func (client *RTMPSClient) Dial(dialurl string) error {
	var c net.Conn
	var err error
	c, err = tls.Dial("tcp", dialurl, nil)
	if err != nil {
		return err
	}
	br := bufio.NewReader(c)
	bw := bufio.NewWriter(c)
	timeout := time.Duration(10) * time.Second
	err = Handshake(c, br, bw, timeout)
	client.conn = c
	client.conn.SetDeadline(time.Now().Add(time.Minute * 30))
	return err
}

func (client *RTMPSClient) Connect() (err error) {
	err = client.Dial(client.addr)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	go writeLoop(client.conn, client.writeChan)
	go messageManager(client.lookupRequestChan, client.storeRequestChan)
	go InvokeIdGenerator(client.idGeneratorChannel)
	connectChan := make(chan connectionResponse, 1)
	go readLoop(client.conn, connectChan, client.storeRequestChan)
	connect := createConnectMessage(client.app, client.swfURL, "rtmps://"+client.addr, client.pageURL)

	client.SafeWrite(connect)
	var response connectionResponse
	select {
	case response = <-connectChan:

	case <-time.After(10 * time.Second):
		return errors.New("Connect Timed out after 30s")
	}
	if !response.Success {
		if response.Err != nil {
			return response.Err
		}

		return errors.New("Connection Failed, error unknown")
	}

	client.dSID = response.ID
	// if err != nil {
	// 	return err
	// }
	// start := time.Now()
	// c, err := client.obConn.Status()
	return err
}

func (client *RTMPSClient) SafeWrite(data []byte) {
	client.writeChan <- data
}

func (client *RTMPSClient) Login(username, password, clientVersion string) (err error) {
	client.ipAddress, err = getIPAddress()
	if err != nil {
		return err
	}
	authToken, err := getAuthToken(username, password, client.serverinfo)
	if err != nil {
		return err
	}

	//	client.login2(username, password, authToken, clientVersion, client.ipAddress.String(), "en_US")
	login1 := createFirstLoginMessage(username, password, authToken, clientVersion, client.ipAddress.String(), "en_US", client.dSID)
	invokeID := client.GetNextID()
	data := encodeMessage(invokeID, login1)
	client.SafeWrite(data)
	login1result, err := client.blockingLookup(invokeID, 10)
	if err != nil {
		return err
	}
	sessionToken, accountID, err := getLogin1Data(login1result.Message)
	if err != nil {
		return err
	}

	encbuf := []byte(strings.ToLower(username) + ":" + sessionToken)
	login2 := wrapBody("auth", 8, base64.StdEncoding.EncodeToString(encbuf), client.dSID)
	login2.ObjectType = "flex.messaging.messages.CommandMessage"
	invokeID = client.GetNextID()
	data = encodeMessage(invokeID, login2)
	client.SafeWrite(data)
	login2result, err := client.blockingLookup(invokeID, 10)
	if err != nil {
		return err
	}
	if data, ok := login2result.Message["data"].(amf.TypedObject); ok {
		if success, ok := data.Object["body"].(string); !ok || success != "success" {
			return errors.New("Login2 Failed")
		}
	} else {
		return errors.New("Login2 Failed")
	}

	err = client.subscribe(accountID)
	if err != nil {
		return err
	}

	go StartHeartbeat(accountID, sessionToken, client.dSID, client.writeChan, client.lookupRequestChan, client.heartbeatStop, client.idGeneratorChannel)
	return nil
}

func (client *RTMPSClient) blockingLookup(invokeID, timeout int) (result LookupResponse, err error) {
	lookup := NewMessageLookup(invokeID)
	client.lookupRequestChan <- lookup
	select {
	case result = <-lookup.ReturnChan:
		return result, nil
	case <-time.After(time.Duration(timeout) * time.Second):
		return result, errors.New("Timed out after 30s")
	}
}

func (client *RTMPSClient) subscribe(accountID int) (err error) {
	subscriptionCodes := []string{"bc", "cn", "gn"}
	for _, code := range subscriptionCodes {
		subscribeBody := wrapBody("messagingDestination", 0, amf.Object{}, client.dSID)
		subscribeBody.ObjectType = "flex.messaging.messages.CommandMessage"
		subscribeBody, err = writeSubscriptionData(subscribeBody, code, accountID)

		invokeID := client.GetNextID()
		data := encodeMessage(invokeID, subscribeBody)
		client.SafeWrite(data)
		subscription := NewMessageLookup(invokeID)
		client.lookupRequestChan <- subscription
		subscriptionresult := <-subscription.ReturnChan

		if result, ok := subscriptionresult.Message["result"].(string); !ok || result != "_result" {
			return errors.New("Subscription Failed")
		}
	}
	return nil
}

func writeSubscriptionData(subscribeBody amf.TypedObject, subscriptionID string, accountID int) (amf.TypedObject, error) {
	if headers, ok := subscribeBody.Object["headers"].(amf.Object); ok {
		headers["DSSubtopic"] = subscriptionID
	} else {
		return subscribeBody, errors.New("Error Setting Header Value In writeSubscriptionData")
	}
	subscribeBody.Object["clientId"] = fmt.Sprintf("%s-%d", subscriptionID, accountID)
	return subscribeBody, nil
}

func getLogin1Data(login1 amf.Object) (sessionToken string, accountId int, err error) {
	if result, ok := login1["result"].(string); ok {
		if result == "_result" {
			if data, ok := login1["data"].(amf.TypedObject); ok {
				if body, ok := data.Object["body"].(amf.TypedObject); ok {
					if sessionToken, ok = body.Object["token"].(string); ok {
						if accountSummary, ok := body.Object["accountSummary"].(amf.TypedObject); ok {
							if accountId, ok := accountSummary.Object["accountId"].(float64); ok {
								return sessionToken, int(accountId), nil
							}
						}
					}
				}
			}
		}
	}
	return "", -1, errors.New(fmt.Sprintf("Login1 Data Not Acceptable %v", login1))
}

func createFirstLoginMessage(username, password, authToken, clientVersion, ipAddress, locale, dsid string) amf.TypedObject {
	LoginMessage := amf.TypedObject{
		Object: amf.Object{
			"username":      username,
			"password":      password,
			"authToken":     authToken,
			"clientVersion": clientVersion,
			"ipAddress":     ipAddress,
			//"locale":             locale,
			"domain": "lolclient.lol.riotgames.com",
			//"operatingSystem":    "goLolClient",
			//"securityAnswer":     nil,
			//"oldPassword":        nil,
			//"partnerCredentials": nil,
		},
		ObjectType: "com.riotgames.platform.login.AuthenticationCredentials",
	}

	return wrapBody("loginService", "login", LoginMessage, dsid)
}

func (client RTMPSClient) BlockingRequest(destination string, operation interface{}, body interface{}, timeout int) (amf.Object, error) {
	request := wrapBody(destination, operation, body, client.dSID)
	invokeID := client.GetNextID()
	data := encodeMessage(invokeID, request)
	client.SafeWrite(data)
	requestResult, err := client.blockingLookup(invokeID, timeout)
	if err != nil {
		return nil, err
	}
	return requestResult.Message, requestResult.Err
}

func wrapBody(destination string, operation interface{}, body interface{}, dsid string) amf.TypedObject {
	uid, err := uuid.NewV4()
	if err != nil {
		fmt.Println(err)
	}
	return amf.TypedObject{
		ObjectType: "flex.messaging.messages.RemotingMessage",
		Object: amf.Object{
			"destination": destination,
			"operation":   operation,
			//"source":      nil,
			//	"timestamp":   float32(0),
			"messageId": uid.String(),
			//	"timeToLive":  float32(0),
			//	"clientId":    nil,
			"headers": amf.Object{
				"DSRequestTimeout": float32(60),
				"DSId":             dsid,
				"DSEndpoint":       "my-rtmps",
			},
			"body": body,
		},
	}
}

func createConnectMessage(app, swfURL, tcUrl, pageUrl string) []byte {
	amfMessage := amf.Object{
		"objectEncoding": 3,
		"app":            app,
		"fpad":           false,
		"flashVer":       "WIN 10,1,85,3",
		"tcUrl":          tcUrl,
		"audioCodecs":    3191,

		"videoFunction": 1,
		"pageUrl":       pageUrl,
		"capabilities":  239,
		"swfUrl":        swfURL,
		"videoCodecs":   252,
	}
	return encodeConnect(amfMessage)

}

func encodeMessage(id int, data interface{}) []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(0x00)
	buf.WriteByte(0x05)
	_, err := amf.WriteDouble(buf, float64(id))
	if err != nil {
		fmt.Println(err)
	}
	buf.WriteByte(0x05)

	buf.WriteByte(0x11)

	_, err = amf.AMF3_WriteValue(buf, data)
	if err != nil {
		fmt.Println(err)
	}

	ret, err := addHeaders(buf.Bytes())
	if err != nil {
		fmt.Println(err)
	}
	return ret
}

func encodeConnect(params amf.Object) []byte {
	buf := new(bytes.Buffer)
	_, err := amf.WriteString(buf, "connect")
	if err != nil {
		fmt.Println(err)
	}
	_, err = amf.WriteDouble(buf, 1)
	if err != nil {
		fmt.Println(err)
	}
	buf.WriteByte(0x11)
	buf.WriteByte(0x09)
	_, err = amf.AMF3_WriteAssociativeArray(buf, params)
	if err != nil {
		fmt.Println(err)
	}
	buf.WriteByte(0x01)
	buf.WriteByte(0x00)
	_, err = amf.WriteString(buf, "nil")
	if err != nil {
		fmt.Println(err)
	}
	_, err = amf.WriteString(buf, "")
	if err != nil {
		fmt.Println(err)
	}
	uid, _ := uuid.NewV4()

	amfCommandMessage := amf.TypedObject{
		ObjectType: "flex.messaging.messages.CommandMessage",
		Object: amf.Object{
			"messageRefType": nil,
			"operation":      5,
			"correlationId":  "",
			"clientId":       nil,
			"destination":    "",
			"messageId":      uid.String(),
			"timestamp":      float32(0),
			"timeToLive":     float32(0),
			"body":           amf.Object{},
			"headers": amf.Object{
				"DSMessagingVersion": float32(1),
				"DSId":               "my-rtmps",
			},
		},
	}

	buf.WriteByte(0x11)
	_, err = amf.AMF3_WriteTypedObject(buf, amfCommandMessage)
	if err != nil {
		fmt.Println(err)
	}

	message, err := addHeaders(buf.Bytes())
	if err != nil {
		fmt.Println(err)
	}
	message[7] = 0x14
	return message
}

func addHeaders(data []byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.WriteByte(0x03)
	// Timestamp
	timediff := uint32((time.Now().Sub(startTime)).Nanoseconds()/int64(1000000)) % MAX_TIMESTAMP
	tmpBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(tmpBuf, timediff)
	m, err := buf.Write(tmpBuf[1:])
	if err != nil {
		return data, err
	}
	n := m
	// Body size
	binary.BigEndian.PutUint32(tmpBuf, uint32(len(data)))
	m, err = buf.Write(tmpBuf[1:])
	if err != nil {
		return data, err
	}
	n += m
	// Content type
	buf.WriteByte(0x11)

	buf.WriteByte(0x00)
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)

	for i := 0; i < len(data); i++ {
		buf.WriteByte(data[i])
		if i%128 == 127 && i != len(data)-1 {
			buf.WriteByte(0xC3)
		}
	}
	return buf.Bytes(), nil

}

func writeLoop(conn net.Conn, messages chan []byte) {
	for {
		message := <-messages
		conn.Write(message)
		//fmt.Printf("%s\n", message)
		//write to network
	}
}

func getIPAddress() (net.IP, error) {

	resp, err := http.Get("http://ll.leagueoflegends.com/services/connection_info")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ipResponse := struct {
		Ip_address string
	}{}
	err = json.Unmarshal(body, &ipResponse)
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(ipResponse.Ip_address)
	if ip == nil {
		return nil, errors.New("Could Not Retrieve IP address")
	}
	return ip, nil
}

func getAuthToken(username, password string, server ServerInfo) (string, error) {
	//TODO: add garena support
	payload := "user=" + username + ",password=" + password
	//query := "payload=" + url.QueryEscape(payload)
	loginQueueUrl := server.LoginQueue
	loginQueueUrl.Path = "/login-queue/rest/queue/authenticate"
	loginQueueJson, err := getLoginQueueJson(loginQueueUrl.String(), payload)
	if err != nil {
		return "", err
	}
	if loginQueueJson["status"] == "FAILED" {
		if s, ok := loginQueueJson["reason"].(string); ok {
			return "", errors.New("getAuthToken: Error Logging In: " + s)
		} else {
			return "", errors.New("getAuthToken: Error Logging In And Could Not Retrieve Reason ")
		}
	}
	if loginQueueJson["status"] == "BUSY" {
		if s, ok := loginQueueJson["reason"].(string); ok {
			return "", errors.New("getAuthToken: Error Logging In. Busy: " + s)
		} else {
			return "", errors.New("getAuthToken: Error Logging In: Busy ")
		}
	}
	var token interface{}
	var ok bool
	if token, ok = loginQueueJson["token"]; !ok {
		rate := 0
		delay := time.Duration(0)
		node, ok := loginQueueJson["node"].(float64)
		if !ok {
			return "", errors.New("getAuthToken: Node is not an float64")
		}
		champ, ok := loginQueueJson["champ"].(string)
		if !ok {
			return "", errors.New("getAuthToken: Champ is not a string")
		}
		if r, ok := loginQueueJson["rate"].(float64); ok {
			rate = int(r)
		} else {
			return "", errors.New("getAuthToken: Rate is not an float64")
		}
		if d, ok := loginQueueJson["delay"].(float64); ok {
			delay = time.Duration(d)
		} else {
			return "", errors.New("getAuthToken: Delay is not an float64")
		}

		id, cur := 0, 0
		tickers, ok := loginQueueJson["tickers"].([]interface{})
		if !ok {
			return "", errors.New("getAuthToken: tickers is not able to be read")
		}
		for _, o := range tickers {
			if ticker, ok := o.(map[string]interface{}); ok {
				tickerNode, ok := ticker["node"].(float64)
				if !ok {
					return "", errors.New("getAuthToken: Ticker Node is not an float64")
				}
				if node == tickerNode {
					if i, ok := ticker["id"].(float64); ok {
						id = int(i)
					} else {
						return "", errors.New("getAuthToken: Ticker id is not an float64")
					}
					if c, ok := ticker["current"].(float64); ok {
						cur = int(c)
					} else {
						return "", errors.New("getAuthToken: Ticker current is not an float64")
					}
					break
				}
			}
		}
		loginQueueUrl.Path = "/login-queue/rest/queue/ticker/"
		nodeString := strconv.FormatInt(int64(node), 10)
		for id-cur > rate {
			fmt.Printf("%s is in a login queue for %s, #%d in line\n", username, server.Name, int(math.Max(float64(id-cur), 1)))
			time.Sleep(delay)
			cur, err = getTickerQueueCurrent(loginQueueUrl.String(), champ, nodeString)
		}
		loginQueueUrl.Path = "/login-queue/rest/queue/authToken/"
		tokenString, err := getLoginQueueToken(loginQueueUrl.String(), username, delay)
		if err != nil {
			return "", err
		}
		return tokenString, nil

	}
	if s, ok := token.(string); ok {
		return s, nil
	} else {
		return "", errors.New("getAuthToken: Token Is Not A String")
	}
}

func getLoginQueueJson(requestURL, payload string) (map[string]interface{}, error) {
	resp, err := http.PostForm(requestURL, url.Values{"payload": {payload}})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	var loginQueueJson map[string]interface{}
	err = json.Unmarshal(body, &loginQueueJson)
	if err != nil {
		return nil, err
	}
	return loginQueueJson, nil
}
func getTickerQueueCurrent(requestURL, champ, node string) (int, error) {
	resp, err := http.Get(requestURL + champ)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	var tickerQueueJson map[string]interface{}
	err = json.Unmarshal(body, &tickerQueueJson)
	if err != nil {
		return -1, err
	}
	if currentHex, ok := tickerQueueJson[node].(string); ok {
		if i, err := strconv.ParseInt(currentHex, 16, 0); err == nil {
			return int(i), nil
		}
	} else {
		return -1, errors.New("getTickerQueueCurrent: node value Is Not A String")
	}
	return -1, err
}

func getLoginQueueToken(requestURL, username string, delay time.Duration) (string, error) {
	for {
		resp, err := http.Get(requestURL + strings.ToLower(username))
		if err != nil {
			return "", err
		}

		if resp.StatusCode != 404 {
			body, err := ioutil.ReadAll(resp.Body)
			var authTokenJson map[string]interface{}
			err = json.Unmarshal(body, &authTokenJson)
			if err != nil {
				return "", err
			}

			if token, ok := authTokenJson["token"].(string); ok {
				return token, nil
			}
		}

		resp.Body.Close()
		time.Sleep(delay / 10)
	}
}
