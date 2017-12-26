package muxtcp

import (
	"bytes"
	"net"

	"github.com/u35s/gmod/lib/gnet"
)

func ListenAccept(mux *MuxTcp, network, addr string) error {
	listener, err := gnet.Listen(network, addr)
	if err != nil {
		return err
	}
	mux.AddErrHandler(func(err error) {
		listener.Close()
	})
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

func AcceptDial(mux *MuxTcp, network, addr string) {
	acceptSession(mux, func(session *MuxTcpSession) {
		conn, err := gnet.Dial("tcp", addr)
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

func NewMuxTcp(conn net.Conn) *MuxTcp {
	muxtcp := &MuxTcp{
		conn: conn, sessions: make(map[uint]*MuxTcpSession),
		sendChan: make(chan *bytes.Buffer, 12), sessionChan: make(chan *MuxTcpSession, 1024),
	}
	muxtcp.run()
	return muxtcp
}
