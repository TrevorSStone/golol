package rtmpsclient

import (
	"time"
)

func StartHeartbeat(accountID int, sessionToken, dsid string, writeChan chan<- []byte, lookupRequestChan chan<- MessageLookup, cancelChan <-chan bool, idGeneratorChannel <-chan int) {

	heartbeats := 1
Loop:
	for {
		heartbeat := wrapBody("loginService", "performLCDSHeartBeat", []interface{}{accountID, sessionToken, heartbeats, time.Now().Format("002 Jan 2 2006 15:04:05 GMTZ")}, dsid)
		invokeID := <-idGeneratorChannel
		data := encodeMessage(invokeID, heartbeat)
		writeChan <- data
		go func() {
			heartbeatlookup := NewMessageLookup(invokeID)
			lookupRequestChan <- heartbeatlookup
			<-heartbeatlookup.ReturnChan

		}()
		select {
		case <-time.Tick(2 * time.Minute):

		case <-cancelChan:
			break Loop
		}

	}
}
