package muxtcp

import (
	"bytes"
	"errors"
	"log"
	"net"
	"sync"
	"time"
)

type sessionAddr struct {
	network string
	str     string
}

func (this *sessionAddr) Network() string { return this.network }
func (this *sessionAddr) String() string  { return this.str }

type MuxTcpSession struct {
	id         uint
	buf        *bytes.Buffer
	muxtcp     *MuxTcp
	recive     chan []byte
	localAddr  net.Addr
	remoteAddr net.Addr

	once sync.Once
	err  error
}

func (this *MuxTcpSession) MuxTcp() *MuxTcp { return this.muxtcp }

func (this *MuxTcpSession) Read(b []byte) (n int, err error) {
	if this.buf.Len() == 0 {
		select {
		case bts, ok := <-this.recive:
			if !ok {
				return 0, this.err
			} else {
				this.buf.Write(bts)
			}
		}
	}
	return this.buf.Read(b)
}

func (this *MuxTcpSession) Write(bts []byte) (n int, err error) {
	if this.muxtcp == nil {
		return 0, this.err
	}
	buf := new(bytes.Buffer)
	head := new(muxTcpPacketHead)
	head.SessionID = uint16(this.id)
	for i := 1; i > 0; {
		tmp := bts
		var cont uint16
		if len(bts) >= muxTcpMaxPacketLen {
			tmp = bts[:muxTcpMaxPacketLen]
			bts = bts[muxTcpMaxPacketLen:]
			cont = muxTcpPacketHeadContFlag
		} else {
			i = 0
		}
		binWrite(buf, head.SessionID|cont)
		binWrite(buf, uint16(len(tmp)))
		buf.Write(tmp)
	}
	this.muxtcp.sendChan <- buf
	return buf.Len(), nil
}

func (this *MuxTcpSession) Close() error {
	return this.close(errors.New("active close"))
}

func (this *MuxTcpSession) close(err error) error {
	ret := errors.New("already closed")
	this.once.Do(func() {
		this.muxtcp.lock.Lock()
		delete(this.muxtcp.sessions, this.id)
		this.muxtcp.lock.Unlock()
		close(this.recive)
		log.Printf("[muxtcp],%v,%v,session %v close,err %v",
			this.muxtcp.conn.LocalAddr(), this.muxtcp.conn.RemoteAddr(), this.id, err)
		ret = nil
		this.muxtcp = nil
	})
	return ret
}

func (this *MuxTcpSession) LocalAddr() net.Addr                { return this.localAddr }
func (this *MuxTcpSession) RemoteAddr() net.Addr               { return this.remoteAddr }
func (this *MuxTcpSession) SetDeadline(t time.Time) error      { return nil }
func (this *MuxTcpSession) SetReadDeadline(t time.Time) error  { return nil }
func (this *MuxTcpSession) SetWriteDeadline(t time.Time) error { return nil }
