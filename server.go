package sourceserver

import (
	"bytes"
	"errors"
	"net"
	"bufio"
	"time"
	"strings"
	"github.com/golang/glog"
)

const (
	A2S_INFO byte = 0x54
	A2S_PLAYER byte = 0x55
)

type Player struct {
	Name string
	Score int32
}

type Server struct {
	IpPort string
	Name string
	Map string
	Folder string
	Game string
	Id int16
	Players int8
	MaxPlayers int8
	Bots int8 
	ServerType string
	Environment string
	Visibility int8
	Vac int8
	Version string
	Keywords string
	Country string
	Mode string
	PlayersArray []Player
}

type ServerRequest struct {
	Header byte
	Payload string
}

type PlayerListRequest struct {
	Header byte
	Challenge int32
}

func (r *PlayerListRequest) Bytes() []byte {
	bytes := make([]byte, 9)

	bytes[0] = 0xFF
	bytes[1] = 0xFF
	bytes[2] = 0xFF
	bytes[3] = 0xFF
	
	bytes[4] = r.Header
	bytes[5] = byte(r.Challenge >> 24)
	bytes[6] = byte(r.Challenge >> 16)
	bytes[7] = byte(r.Challenge >> 8)
	bytes[8] = byte(r.Challenge)
	
	return bytes
}

func (r *ServerRequest) Bytes() []byte {
	bytes := make([]byte, 6 + len(r.Payload))
	bytes[0] = 0xFF
	bytes[1] = 0xFF
	bytes[2] = 0xFF
	bytes[3] = 0xFF
	bytes[4] = r.Header
	
 	lastIdx := 5
	
	for i := 0; i < len(r.Payload); i++ {
		bytes[lastIdx] = byte(r.Payload[i])
		lastIdx++
	}
	
	bytes[lastIdx] = 0x00
	lastIdx++
	
	return bytes
}

func (s *Server) GetChallenge() int32 {
	serverRequest := PlayerListRequest{A2S_PLAYER, -1}
	
	conn, err := net.Dial("udp", s.IpPort)
	defer conn.Close()
	
	conn.SetReadDeadline(time.Now().Add(time.Second))

	var size int
	newChallenge := int32(-1)
	
	if err == nil {
		conn.Write(serverRequest.Bytes())

		p :=  make([]byte, 2048)
		size, err = bufio.NewReader(conn).Read(p)

		p = p[0:size]
		if err == nil {
			if bytes.Compare([]byte {0xFF, 0xFF, 0xFF, 0xFF, 0x41}, p[0:5]) == 0 {
				_, newChallenge = DecodeInt32(p, 5)	
			}
		} else {
			glog.Errorf("Error reading challenge response from %v. Reason: %v\n", s.IpPort, err)
		}
	}

	return newChallenge
}

func (s *Server) GetPlayerList(newChallenge int32) {
	serverRequest := PlayerListRequest{A2S_PLAYER, newChallenge}
	conn, err := net.Dial("udp", s.IpPort)
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(time.Second))

	var size int

	if err == nil {
		req := serverRequest.Bytes()
		conn.Write(req)

		p :=  make([]byte, 2048)
		size, err = bufio.NewReader(conn).Read(p)

		p = p[0:size]
		if err == nil {
			err = s.DecodePlayers(p)
			if (err != nil) {
				glog.Errorf("Error Decoding players info from %v. Reason: %v\n", s.IpPort, err)
			}
		} else {
			glog.Errorf("Error reading players info from %v. Reason: %vv\n", s.IpPort, err)
		}
	} else {
		glog.Errorf("Error querying server for player info from %v. Reason: %v\n", s.IpPort, err)
	}
}

func (s *Server) GetPlayersInfo() {
	newChallenge := s.GetChallenge()
	if newChallenge != 0 {
		s.GetPlayerList(newChallenge)
	}
}

func (s *Server) GetServerInfo() bool {
	serverRequest := ServerRequest{A2S_INFO, "Source Engine Query"}

	conn, err := net.Dial("udp", s.IpPort)
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(time.Second))

	var size int

	if err == nil {
		conn.Write(serverRequest.Bytes())

		p :=  make([]byte, 2048)
		size, err = bufio.NewReader(conn).Read(p)

		p = p[0:size]
		if err == nil {
			if (s.Decode(p) != nil) {
				glog.Errorf("Error decoding server info from %v. Reason: %v\n", s.IpPort, err)
			} else {
				return true
			}
		} else {
			glog.Errorf("Error reading server info from %v. Reason: %v\n", s.IpPort, err)
		}
	} else {
		glog.Errorf("Error querying server info from %v. Reason: %v\n", s.IpPort, err)
	}
	return false
}

func (s *Server) DecodePlayers(b []byte) error {
	
	expectedHeader := []byte {0xFF, 0xFF, 0xFF, 0xFF, 0x44}
	
	if bytes.Compare(expectedHeader, b[0:5]) != 0 {
		return errors.New("Header error")
	}
	
	numberOfPlayers := int(b[5])
	
	currIndex := 6
		
	var name string
	var score int32
	for i := 0; i < numberOfPlayers; i++ {
		//ignoring Index for now
		currIndex++
		currIndex, name = DecodeString(b, currIndex)
		currIndex, score = DecodeInt32LittleEndian(b, currIndex)
		//Ignoring Duration
		currIndex += 4
		
		s.PlayersArray = append(s.PlayersArray, Player{Name:name, Score:score})
	}
	
	return nil
}


func (s *Server) Decode(b []byte) error {
	
	expectedHeader := []byte { 0xFF, 0xFF, 0xFF, 0xFF, 0x49 }
	
	if bytes.Compare(b[0:5], expectedHeader) != 0 {
		return errors.New("Header error")
	}
	
	s.PlayersArray = []Player{}
	
	currIndex := 6
		
	currIndex, s.Name = DecodeString(b, currIndex)
	currIndex, s.Map = DecodeString(b, currIndex)
	currIndex, s.Folder = DecodeString(b, currIndex)
	currIndex, s.Game = DecodeString(b, currIndex)
	
	s.Id = int16(b[currIndex] << 8) | int16(b[currIndex+1])
	currIndex += 2
	
	s.Players = int8(b[currIndex])
	currIndex++
	
	s.MaxPlayers = int8(b[currIndex])
	currIndex++
	
	s.Bots = int8(b[currIndex])
	currIndex++
	
	s.ServerType = string(b[currIndex])
	currIndex++
	
	s.Environment = string(b[currIndex])
	currIndex++
	
	s.Visibility = int8(b[currIndex])
	currIndex++
	
	s.Vac = int8(b[currIndex])
	currIndex++
	
	currIndex, s.Version = DecodeString(b,currIndex)
	
	extraFlag := b[currIndex]
	currIndex++
	
	if extraFlag & 0x80 > 0 {
		//ignore this
		currIndex += 2
	}
	if extraFlag & 0x10 > 0 {
		//ignore this
		currIndex += 8
	}
	if extraFlag & 0x40 > 0 {
		currIndex += 2
		currIndex, _ = DecodeString(b, currIndex)
	}
	if extraFlag & 0x20 > 0 {
		currIndex, s.Keywords = DecodeString(b, currIndex)
	}

	// not serialized, but read after serialization is done
	KeyWords := strings.Split(s.Keywords, "|")
	if (len(KeyWords) >= 2) {
		s.Mode = KeyWords[0]
		s.Country = KeyWords[1]	
	}	
	return nil
}