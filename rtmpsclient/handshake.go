package rtmpsclient

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"time"
)

const (
	RIOT_SIG_SIZE          = 1528
	RTMP_SIG_SIZE          = 1536
	RTMP_LARGE_HEADER_SIZE = 12
	SHA256_DIGEST_LENGTH   = 32
	RTMP_DEFAULT_CHUNKSIZE = 128
	MAX_TIMESTAMP          = uint32(2000000000)
)

func Handshake(c net.Conn, br *bufio.Reader, bw *bufio.Writer, timeout time.Duration) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	// Send C0+C1
	err = bw.WriteByte(0x03)
	CheckError(err, "Handshake() Send C0")

	c1 := CreateRandomBlock(RTMP_SIG_SIZE)
	// Set Timestamp
	binary.BigEndian.PutUint32(c1, uint32(GetTimestamp()))
	binary.BigEndian.PutUint32(c1[4:], uint32(0))
	_, err = bw.Write(c1)
	CheckError(err, "Handshake() Send C1")
	if timeout > 0 {
		c.SetWriteDeadline(time.Now().Add(timeout))
	}
	err = bw.Flush()
	CheckError(err, "Handshake() Flush C0+C1")

	// Read S0
	if timeout > 0 {
		c.SetReadDeadline(time.Now().Add(timeout))
	}
	s0, err := br.ReadByte()
	CheckError(err, "Handshake() Read S0")
	if s0 != 0x03 {
		return errors.New(fmt.Sprintf("Handshake() Got S0: %x", s0))
	}

	// Read S1
	s1 := make([]byte, RTMP_SIG_SIZE)
	if timeout > 0 {
		c.SetReadDeadline(time.Now().Add(timeout))
	}
	_, err = io.ReadAtLeast(br, s1, RTMP_SIG_SIZE)
	CheckError(err, "Handshake Read S1")
	c2 := CreateRandomBlock(RTMP_SIG_SIZE)
	for i := 0; i < 4; i++ {
		c2[i] = s1[i]
	}
	binary.BigEndian.PutUint32(c2[4:], uint32(GetTimestamp()))
	for i := 8; i < RTMP_SIG_SIZE; i++ {
		c2[i] = s1[i]
	}
	// Send C2
	_, err = bw.Write(c2)
	CheckError(err, "Handshake() Send C2")
	if timeout > 0 {
		c.SetWriteDeadline(time.Now().Add(timeout))
	}
	err = bw.Flush()
	CheckError(err, "Handshake() Flush C2")

	// Read S2

	if timeout > 0 {
		c.SetReadDeadline(time.Now().Add(timeout))
	}
	s2 := make([]byte, RTMP_SIG_SIZE)
	_, err = io.ReadAtLeast(br, s2, RTMP_SIG_SIZE)
	CheckError(err, "Handshake() Read S2")

	valid := bytes.Compare(c1, s2) == 0
	if !valid {
		return errors.New("Handshake C1 and S2 are not a match")
	}
	return err
}

// Get timestamp
func GetTimestamp() uint32 {
	//return uint32(0)
	return uint32(time.Now().UnixNano()/int64(1000000)) % MAX_TIMESTAMP
}

func CreateRandomBlock(size uint) []byte {
	/*
		buf := make([]byte, size)
		for i := uint(0); i < size; i++ {
			buf[i] = byte(rand.Int() % 256)
		}
		return buf
	*/

	size64 := size / uint(8)
	buf := new(bytes.Buffer)
	var r64 int64
	var i uint
	for i = uint(0); i < size64; i++ {
		r64 = rand.Int63()
		binary.Write(buf, binary.BigEndian, &r64)
	}
	for i = i * uint(8); i < size; i++ {
		buf.WriteByte(byte(rand.Int()))
	}
	return buf.Bytes()

}
