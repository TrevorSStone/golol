package rtmpsclient

import (
	"fmt"
	"github.com/TrevorSStone/goamf"
	"strconv"
)

type MessageLookup struct {
	MessageID  int
	ReturnChan chan LookupResponse
}

type LookupResponse struct {
	Err     error
	Message amf.Object
}

func NewMessageLookup(id int) MessageLookup {
	return MessageLookup{
		MessageID:  id,
		ReturnChan: make(chan LookupResponse),
	}
}

func messageManager(lookupRequestChan <-chan MessageLookup, storeRequestChan <-chan amf.Object) {
	messageMap := make(map[int]amf.Object)
	channelMap := make(map[int]chan chan LookupResponse)
	for {
		select {
		case lookupRequest := <-lookupRequestChan:
			if message, ok := messageMap[lookupRequest.MessageID]; ok {
				lookupRequest.ReturnChan <- LookupResponse{Message: message}
				delete(messageMap, lookupRequest.MessageID)
			} else {

				if channel, ok := channelMap[lookupRequest.MessageID]; ok {
					go func() {
						channel <- lookupRequest.ReturnChan
					}()
				} else {
					channelMap[lookupRequest.MessageID] = make(chan chan LookupResponse)
					go func() {
						channelMap[lookupRequest.MessageID] <- lookupRequest.ReturnChan
					}()
				}
				//lookupRequest.ReturnChan <- LookupResponse{Err: errors.New("MessageID not found")}
			}

		case storeRequest := <-storeRequestChan:
			if id, ok := storeRequest["invokeId"]; ok {
				invokeId := -1
				switch id := id.(type) {
				default:
					fmt.Printf("unexpected type %T for invokeId", id)
				case float64:
					invokeId = int(id)
				case float32:
					invokeId = int(id)
				case int:
					invokeId = id

				case string:
					var err error
					invokeId, err = strconv.Atoi(id)
					if err != nil {
						fmt.Printf("unexpected type %T for invokeId", id)
					}

				}
				if invokeId > -1 {
					if returnChans, ok := channelMap[invokeId]; ok {
					Loop:
						for {
							select {
							case returnChan := <-returnChans:
								returnChan <- LookupResponse{Message: storeRequest}
							default:
								delete(channelMap, invokeId)
								break Loop
							}

						}
					} else {
						messageMap[invokeId] = storeRequest
					}
				}

			} else {
				fmt.Println("Tried to store message with invalid invokeId")
			}
		}
	}
}
