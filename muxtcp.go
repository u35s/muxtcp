package muxtcp

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"sync"
)

const (
	muxTcpMaxPacketLen                   = 0xffff
	muxTcpPacketHeadLen                  = 4
	muxTcpPacketHeadContFlag      uint16 = 1 << 15 //连续
	muxTcpPacketHeadSessionIDMask        = 32767   //正常session id的mask
)

type muxTcpPacketHead struct {
	SessionID  uint16
	PacketSize uint16
}

type muxTcpPacketRead struct {
	muxTcpPacketHead
	Cont    bool //是否连续
	HasHead bool //是否已读
	Buf     *bytes.Buffer
}

func (this *muxTcpPacketRead) parse() {
	this.Cont = (this.SessionID & muxTcpPacketHeadContFlag) > 0
	this.SessionID = this.SessionID & muxTcpPacketHeadSessionIDMask
}

const (
	flowTypeRev = iota
	flowTypeSend
	flowTypeNum
)

type MuxTcp struct {
	uid         uint
	conn        net.Conn
	flow        [flowTypeNum]int
	sessions    map[uint]*MuxTcpSession
	sendChan    chan *bytes.Buffer
	sessionChan chan *MuxTcpSession

	lock sync.RWMutex

	once       sync.Once
	errHandler func(error)
}

func (this *MuxTcp) Open(id uint) *MuxTcpSession {
	if id == 0 {
		if this.uid < 10 || this.uid >= 32767 {
			this.uid = 10
		}
		this.uid++
		id = this.uid
	}
	this.lock.RLock()
	session, ok := this.sessions[id]
	if ok {
		this.lock.RUnlock()
		return session
	}
	this.lock.RUnlock()
	this.lock.Lock()
	session = &MuxTcpSession{id: id, muxtcp: this,
		localAddr: &sessionAddr{network: this.conn.LocalAddr().Network(),
			str: fmt.Sprintf("%v-%v", this.conn.LocalAddr().String(), id)},
		remoteAddr: this.conn.RemoteAddr(),
		recive:     make(chan []byte, 12), buf: new(bytes.Buffer)}
	this.sessions[session.id] = session
	this.lock.Unlock()
	return session
}

func (this *MuxTcp) drawPacket(buf *bytes.Buffer, read *muxTcpPacketRead) {
	for {
		if !read.HasHead {
			if buf.Len() < muxTcpPacketHeadLen {
				break
			}
			binRead(buf, &read.SessionID)
			binRead(buf, &read.PacketSize)
			read.HasHead = true
			read.parse()
		}
		if buf.Len() < int(read.PacketSize) {
			break
		}
		read.HasHead = false
		data := buf.Next(int(read.PacketSize))
		if read.Cont {
			if read.Buf == nil {
				read.Buf = &bytes.Buffer{}
			}
			read.Buf.Write(data)
			continue
		} else if read.Buf != nil && read.Buf.Len() > 0 {
			read.Buf.Write(data)
			data = read.Buf.Bytes()
			read.Buf = nil
		}
		this.lock.RLock()
		session, ok := this.sessions[uint(read.SessionID)]
		this.lock.RUnlock()
		if !ok {
			session = this.Open(uint(read.SessionID))
			this.sessionChan <- session
		}
		session.recive <- data
	}
}

func (this *MuxTcp) sendPacket() error {
	for {
		select {
		case b := <-this.sendChan:
			if b == nil {
				return errors.New("send is nil")
			}
			this.flow[flowTypeSend] += b.Len()
			_, err := this.conn.Write(b.Bytes())
			if err != nil {
				return err
			}
		}
	}
}

func (this *MuxTcp) recivePacket() error {
	var buf bytes.Buffer
	var read muxTcpPacketRead
	bts := make([]byte, 1<<16)
	for {
		num, err := this.conn.Read(bts)
		if err != nil {
			return err
		}
		this.flow[flowTypeRev] += num
		buf.Write(bts[:num])
		this.drawPacket(&buf, &read)
	}
}

func (this *MuxTcp) run() {
	go func() {
		this.handleError(this.recivePacket(), func() {
			this.sendChan <- nil
			close(this.sessionChan)
		})
	}()
	go func() {
		this.handleError(this.sendPacket(), func() {
			this.conn.Close()
			close(this.sessionChan)
		})
	}()
}

func (this *MuxTcp) handleError(err error, end func()) {
	this.once.Do(func() {
		if this.errHandler != nil {
			this.errHandler(err)
		}
		if end != nil {
			end()
		}
	})
}

func (this *MuxTcp) Close(err error) {
	this.handleError(err, func() {
		for _, s := range this.sessions {
			s.close(err)
		}
		close(this.sessionChan)
		this.sendChan <- nil
		this.conn.Close()
	})
}
