package protocol

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

type Transport struct {
	conn     *net.UDPConn
	handlers map[uint8]MessageHandler
	pending  map[uint32]*pendingRequest
	seqNum   uint32
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

type MessageHandler func(*Message, *net.UDPAddr) error

type pendingRequest struct {
	msg      *Message
	addr     *net.UDPAddr
	respChan chan *Message
	timer    *time.Timer
	attempts int
}

type TransportConfig struct {
	LocalAddr string
	N1        int
	T1        time.Duration
}

func NewTransport(cfg *TransportConfig) (*Transport, error) {
	addr, err := net.ResolveUDPAddr("udp", cfg.LocalAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	t := &Transport{
		conn:     conn,
		handlers: make(map[uint8]MessageHandler),
		pending:  make(map[uint32]*pendingRequest),
		seqNum:   1,
		ctx:      ctx,
		cancel:   cancel,
	}

	t.wg.Add(1)
	go t.receiveLoop()

	return t, nil
}

func (t *Transport) Close() error {
	t.cancel()
	t.conn.Close()
	t.wg.Wait()
	return nil
}

func (t *Transport) RegisterHandler(msgType uint8, handler MessageHandler) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.handlers[msgType] = handler
}

func (t *Transport) SendRequest(msg *Message, addr *net.UDPAddr, timeout time.Duration, maxRetries int) (*Message, error) {
	msg.Header.SequenceNumber = t.nextSeqNum()

	req := &pendingRequest{
		msg:      msg,
		addr:     addr,
		respChan: make(chan *Message, 1),
		attempts: 0,
	}

	t.mu.Lock()
	t.pending[msg.Header.SequenceNumber] = req
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		delete(t.pending, msg.Header.SequenceNumber)
		t.mu.Unlock()
	}()

	if err := t.send(msg, addr); err != nil {
		return nil, err
	}

	retryTimer := time.NewTimer(timeout)
	defer retryTimer.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return nil, fmt.Errorf("transport closed")

		case resp := <-req.respChan:
			return resp, nil

		case <-retryTimer.C:
			req.attempts++
			if req.attempts >= maxRetries {
				return nil, fmt.Errorf("max retries exceeded")
			}

			if err := t.send(msg, addr); err != nil {
				return nil, err
			}

			retryTimer.Reset(timeout)
		}
	}
}

func (t *Transport) SendResponse(msg *Message, addr *net.UDPAddr) error {
	return t.send(msg, addr)
}

func (t *Transport) send(msg *Message, addr *net.UDPAddr) error {
	data, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	_, err = t.conn.WriteToUDP(data, addr)
	return err
}

func (t *Transport) receiveLoop() {
	defer t.wg.Done()

	buf := make([]byte, 65535)

	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		t.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, addr, err := t.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if t.ctx.Err() != nil {
				return
			}
			continue
		}

		msg := &Message{}
		if err := msg.Unmarshal(buf[:n]); err != nil {
			continue
		}

		t.handleMessage(msg, addr)
	}
}

func (t *Transport) handleMessage(msg *Message, addr *net.UDPAddr) {
	t.mu.RLock()
	pending, isPending := t.pending[msg.Header.SequenceNumber]
	t.mu.RUnlock()

	if isPending {
		select {
		case pending.respChan <- msg:
		default:
		}
		return
	}

	t.mu.RLock()
	handler, ok := t.handlers[msg.Header.MessageType]
	t.mu.RUnlock()

	if ok {
		handler(msg, addr)
	}
}

func (t *Transport) nextSeqNum() uint32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	seq := t.seqNum
	t.seqNum++
	if t.seqNum > 0xFFFFFF {
		t.seqNum = 1
	}
	return seq
}
