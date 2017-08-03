package muxtcp

import (
	"bytes"
	"net"
	"time"
)

func ListenAccept(mux *MuxTcp, addr string) error {
	listener, err := listen(addr)
	if err != nil {
		return err
	}
	acceptConn(listener, func(conn net.Conn) {
		session := mux.Open(0)
		go func() {
			bts := make([]byte, 1<<16)
			for {
				if n, err := conn.Read(bts); err != nil {
					session.close(err)
					break
				} else {
					session.Write(bts[:n])
				}
			}
		}()
		go func() {
			bts := make([]byte, 1<<16)
			for {
				if n, err := session.Read(bts); err != nil {
					conn.Close()
					break
				} else if _, err = conn.Write(bts[:n]); err != nil {
					session.close(err)
					break
				}
			}
		}()
	})
	return nil
}

func AcceptDial(mux *MuxTcp, addr string) {
	acceptSession(mux, func(session *MuxTcpSession) {
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			return
		}
		go func() {
			bts := make([]byte, 1<<16)
			for {
				if n, err := session.Read(bts); err != nil {
					conn.Close()
					break
				} else if _, err = conn.Write(bts[:n]); err != nil {
					session.close(err)
				}
			}
		}()
		go func() {
			bts := make([]byte, 1<<16)
			for {
				if n, err := conn.Read(bts); err != nil {
					session.close(err)
					break
				} else {
					session.Write(bts[:n])
				}
			}
		}()
	})
}

func NewMuxTcp(conn net.Conn, errHandler func(error)) *MuxTcp {
	muxtcp := &MuxTcp{
		conn: conn, sessions: make(map[uint]*MuxTcpSession),
		sendChan: make(chan *bytes.Buffer, 12), sessionChan: make(chan *MuxTcpSession, 1024),
		errHandler: errHandler,
	}
	muxtcp.run()
	return muxtcp
}
