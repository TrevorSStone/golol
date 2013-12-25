package rtmpsclient

import (
	"bytes"
	"fmt"
	"github.com/TrevorSStone/goamf"
	"net"
	"strconv"
	"time"
)

type packet struct {
	dataBuffer  []byte
	dataPos     int
	dataSize    int
	messageType int
}

func (pack *packet) SetSize(size int) {
	pack.dataSize = size
	pack.dataBuffer = make([]byte, size)
	//pack.dataPos = 0
}
func (pack *packet) SetType(t int) {
	pack.messageType = t
}
func (pack *packet) Add(b byte) {
	pack.dataBuffer[pack.dataPos] = b
	pack.dataPos++
}
func (pack *packet) IsComplete() bool {
	return pack.dataPos == pack.dataSize
}
func (pack *packet) Size() int {
	return pack.dataSize
}
func (pack *packet) Type() int {
	return pack.messageType
}
func (pack *packet) Data() []byte {
	return pack.dataBuffer
}

func (pack *packet) Pos() int {
	return pack.dataPos
}

func readLoop(conn net.Conn, connectionChan chan<- connectionResponse, storeRequestChan chan<- amf.Object) {
	packets := make(map[int]*packet)

	for {
		basicHeader, err := readByte(conn)
		if err != nil {
			fmt.Println(err)
			time.Sleep(1 * time.Second)
			continue
		}
		channel := int(basicHeader & 0x2F)
		headerType := basicHeader & 0xC0
		headerSize := 0
		if headerType == 0x00 {
			headerSize = 12
		} else if headerType == 0x40 {
			headerSize = 8
		} else if headerType == 0x80 {
			headerSize = 4
		} else if headerType == 0xC0 {
			headerSize = 1
		}
		p := &packet{}
		if pack, ok := packets[channel]; ok {
			p = pack

		} else {
			packets[channel] = p
		}

		if headerSize > 1 {
			//TODO: Does this need the -1?
			header := make([]byte, headerSize-1)
			for i := 0; i < len(header); i++ {
				header[i], err = readByte(conn)
				if err != nil {
					fmt.Println(err)
					continue
				}
			}
			if headerSize >= 8 {
				size := 0
				for i := 3; i < 6; i++ {
					size = size*256 + int(header[i]&0xFF)
				}
				p.SetSize(size)
				p.SetType(int(header[6]))
			}

		}

		for i := 0; i < 128; i++ {
			b, err := readByte(conn)
			if err != nil {
				fmt.Println(err)
			}
			p.Add(b)
			if p.IsComplete() {

				break
			}
		}

		if !p.IsComplete() {
			continue
		}
		delete(packets, channel)

		switch p.Type() {
		case 0x14:
			response := DecodeConnect(p.Data())
			connectionChan <- response
		case 0x11:
			response := DecodeMessage(p.Data())
			storeRequestChan <- response
		case 0x06:
			data := p.Data()
			windowSize := 0
			for i := 0; i < 4; i++ {
				windowSize = windowSize*256 + int(data[i]&0xFF)
			}
		case 0x03:

		default:
			fmt.Println("Unrecognized message Type")
		}

		//fmt.Println(p)
	}
	fmt.Println("broke")
}

func readByte(conn net.Conn) (byte, error) {
	buf := make([]byte, 1)
	_, err := conn.Read(buf)
	if err != nil {
		return 0x00, err
	}
	return buf[0], err
}

func DecodeMessage(data []byte) amf.Object {
	r := bytes.NewReader(data)
	decoder := amf.AMF3Decoder{}
	result := amf.Object{}
	var err error
	if data[0] == 0x00 {
		result["version"], err = r.ReadByte()
		if err != nil {
			fmt.Println(err)
		}
	}
	result["result"], err = decoder.ReadValue(r)
	if err != nil {
		fmt.Println(err)
	}
	result["invokeId"], err = decoder.ReadValue(r)
	if err != nil {
		fmt.Println(err)
	}
	result["serviceCall"], err = decoder.ReadValue(r)
	if err != nil {
		fmt.Println(err)
	}
	result["data"], err = decoder.ReadValue(r)
	if err != nil {
		fmt.Println(err)
	}
	i := int64(0)
	for {
		i++
		result[strconv.FormatInt(i, 16)], err = decoder.ReadValue(r)
		if err != nil {
			break
		}
	}
	return result
}

func DecodeConnect(data []byte) connectionResponse {
	r := bytes.NewReader(data)
	response := connectionResponse{}
	var Objects []interface{}
	var err error
	name, err := amf.ReadString(r)
	if err != nil {
		fmt.Println("AMF0 Read name err:", err)
		response.Err = err
		return response
	}
	if name != "_result" {
		return response
	}
	_, err = amf.ReadDouble(r)
	if err != nil {
		fmt.Println("AMF0 Read transactionID err:", err)
		response.Err = err
		return response
	}
	for r.Len() > 0 {
		object, err := amf.ReadValue(r)
		if err != nil {
			fmt.Println("AMF0 Read object err:", err)
			response.Err = err
			return response
		}
		Objects = append(Objects, object)

	}
	if id, ok := Objects[1].(amf.Object)["id"]; ok {
		if idstring, ok := id.(string); ok {
			response.ID = idstring
			response.Success = true
		}
	}
	return response
}
