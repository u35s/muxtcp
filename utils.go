package muxtcp

import (
	"bytes"
	"encoding/binary"
	"net"
)

func binRead(buf *bytes.Buffer, data interface{}) {
	binary.Read(buf, binary.LittleEndian, data)
}

func binWrite(buf *bytes.Buffer, data interface{}) {
	binary.Write(buf, binary.LittleEndian, data)
}

func listen(addr string) (*net.TCPListener, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", addr)
	if err != nil {
		return nil, err
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return nil, err
	}
	return listener, nil
}

func acceptConn(listener net.Listener, f func(conn net.Conn)) {
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				break
			}
			go f(conn)
		}
	}()
}

func acceptSession(muxtcp *MuxTcp, f func(*MuxTcpSession)) {
	go func() {
		for {
			select {
			case session, ok := <-muxtcp.sessionChan:
				if !ok {
					return
				}
				go func() {
					f(session)
				}()
			}
		}
	}()
}
