package sourceserver

import (
	"bytes"
	"errors"
	"github.com/golang/glog"
	"time"
	"net"
	"bufio"
	"fmt"
)

type MasterRequest struct {
	Message byte
	RegionCode byte
	IpPort string
	Filter string
}

type MasterQuery struct {
	RegionCode byte
	AppId int
}

func ReadFromIO(conn net.Conn) []byte {
	p :=  make([]byte, 2046)
	size, err := bufio.NewReader(conn).Read(p)
	p = p[0:size]

	if err == nil {
		if size == 2046 {
			p = append(p, ReadFromIO(conn)...)
		}
		return p
	} else {
		glog.Errorf("Error readying master server response %v\n", err)
		return nil
	}
}

func ReadServers(IpPort string, Filter string) []Server {
	conn, err := net.Dial("udp", "hl2master.steampowered.com:27011")
	defer conn.Close()

	if err != nil {
		glog.Errorf("Error querying master server %v\n", err)
		return nil
	}

	conn.SetReadDeadline(time.Now().Add(time.Second))

	request := MasterRequest{0x31, 0xFF, IpPort, Filter}

	conn.Write(request.Bytes())

	servers, isLast, err := Decode(ReadFromIO(conn))
	if !isLast && err == nil {
		fmt.Printf("Not last\n")
		servers = append(servers, ReadServers(servers[len(servers)-1].IpPort, Filter)...)
	}
	fmt.Printf("Is last")
	return servers;
}

func (m *MasterQuery) Query() []Server {
	servers := ReadServers("0.0.0.0", m.FormatFilter())

	glog.Infof("Master server returned %v servers\n", len(servers))

	for i := 0; i < len(servers); i++ {
		fmt.Printf("Server ip %v\n", servers[i].IpPort)
		if !servers[i].GetServerInfo() || servers[i].MaxPlayers == 0 {
			glog.Infof("Ignoring %v\n", servers[i].IpPort)
			servers = append(servers[:i], servers[i+1:]...)
			i--
		} else if servers[i].Players > 0 {
			servers[i].GetPlayersInfo()
		}
	}

	return servers
}

func (m *MasterQuery) FormatFilter() string {
	Filter := ""
	if (m.AppId > 0) {
		Filter = Filter + fmt.Sprintf("\\appid\\%v", m.AppId)
	}
	return Filter
}

func (r *MasterRequest) Bytes() []byte {
	bytes := make([]byte, 4 + len(r.IpPort) + len(r.Filter))
	bytes[0] = r.Message
	bytes[1] = r.RegionCode
	
	lastIdx := 2
	
	for i := 0; i < len(r.IpPort); i++ {
		bytes[lastIdx + i] = byte(r.IpPort[i])
	}
	
	bytes[lastIdx + len(r.IpPort)] = 0x00
	lastIdx += len(r.IpPort) + 1
	
	for i := 0; i < len(r.Filter); i++ {
		bytes[lastIdx + i] = byte(r.Filter[i])
	}
	bytes[lastIdx + len(r.Filter)] = 0x00
	
	return bytes
}

func Decode(b []byte) ([]Server, bool, error) {
	var servers []Server
	
	length := 6
	expectedHeader := []byte { 0xFF, 0xFF, 0xFF, 0xFF, 0x66, 0x0A }

	if (len(b) < 12) {
		return nil, false, errors.New("Buffer too small")
	}

	var isLast bool
	
	if bytes.Compare(b[0:6], expectedHeader) != 0 {
		return nil, false, errors.New("Header error")
	}
	
	for i := length; i<len(b); i = i+length {
		a1 := b[i]
		a2 := b[i+1]
		a3 := b[i+2]
		a4 := b[i+3]
		port := uint16(b[i+4]) << 8 | uint16(b[i+5])
		
		if a1 == 0 && a2 == 0 && a2 == 0 && a3 == 0 {
			isLast = true
			break
		}
		servers = append(servers, Server{IpPort : fmt.Sprintf("%d.%d.%d.%d:%d", a1, a2, a3, a4, port)})
	}
	
	return servers, isLast, nil
}
