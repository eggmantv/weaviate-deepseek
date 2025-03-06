package connpool

import (
	"context"
	"sync"

	"github.com/gorilla/websocket"
)

type BroadcastFunc func(*Pool, []byte)
type OnAddFunc func(gid string, c *Client)
type OnRemoveFunc func(gid string)

// Clients keep all connections
type Pool struct {
	Clients      map[string]*Client
	ClientsGroup map[string][]string
	lock         sync.Mutex

	broadcastFunc BroadcastFunc

	OnAdd    OnAddFunc
	OnRemove OnRemoveFunc
}

func NewPool() *Pool {
	return &Pool{
		Clients:      make(map[string]*Client),
		ClientsGroup: make(map[string][]string),
		lock:         sync.Mutex{},
	}
}

func (p *Pool) Add(gid string, c *Client) {
	p.lock.Lock()
	defer p.lock.Unlock()

	// spew.Dump("params:", c)
	p.Clients[c.ConnID] = c
	if _, found := p.ClientsGroup[gid]; found {
		p.ClientsGroup[gid] = append(p.ClientsGroup[gid], c.ConnID)
	} else {
		p.ClientsGroup[gid] = []string{c.ConnID}
	}

	if p.OnAdd != nil {
		go p.OnAdd(gid, c)
	}
}

func (p *Pool) Remove(gid, sessionID string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if _, found := p.Clients[sessionID]; found {
		// close(p.Clients[sessionID].SendChan)
		delete(p.Clients, sessionID)

		if p.OnRemove != nil {
			go p.OnRemove(gid)
		}
	}

	if _, exists := p.ClientsGroup[gid]; exists {
		cs := make([]string, 0)
		for _, sid := range p.ClientsGroup[gid] {
			if sid != sessionID {
				cs = append(cs, sid)
			}
		}

		if len(cs) > 0 {
			p.ClientsGroup[gid] = cs
		} else {
			delete(p.ClientsGroup, gid)
		}
	}
}

func (p *Pool) OnBroadcast(f BroadcastFunc) {
	p.broadcastFunc = f
}

func (p *Pool) Broadcast(data []byte) {
	p.broadcastFunc(p, data)
}

// Conn connection object
type Client struct {
	GID    string
	ConnID string
	Conn   *websocket.Conn
	Params interface{}

	SendChan  chan []byte
	CloseChan chan string

	// internal message handler context
	ChatCtx      context.Context
	ChatCancelFn context.CancelFunc
}

func NewClient(gid string, sessionID string, req interface{}) *Client {
	return &Client{
		GID:    gid,
		ConnID: sessionID,
		Params: req,

		SendChan:  make(chan []byte, 2),
		CloseChan: make(chan string, 1),
	}
}
