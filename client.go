package godriver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"
)

type PeerID string

type Message struct {
	Type      string `json:"type"`
	To        PeerID `json:"to"`
	From      PeerID `json:"from"`
	Msg       string `json:"msg"`
	ID        string `json:"id"`
	Intercept bool   `json:"intercept"`
}

type ClientController struct {
	directiveHandler DirectiveHandler
	peerID           PeerID
	masterAddr       string
	listenAddr       string
	peerInfo         map[string]interface{}
	server           *http.Server
	fromNode         chan *Message
	toNode           chan *Message
	resetting        bool
	resettingLock    *sync.Mutex
	stopCh           chan bool

	intercepting     bool
	interceptingLock *sync.Mutex

	timer *timer

	ready     bool
	readyLock *sync.Mutex
}

func NewClientController(peerID PeerID, masterAddr, listenAddr string, peerInfo map[string]interface{}, directiveHandler DirectiveHandler) *ClientController {
	c := &ClientController{
		peerID:           peerID,
		masterAddr:       masterAddr,
		listenAddr:       listenAddr,
		peerInfo:         peerInfo,
		fromNode:         make(chan *Message, 10),
		toNode:           make(chan *Message, 10),
		resetting:        false,
		resettingLock:    new(sync.Mutex),
		stopCh:           make(chan bool),
		directiveHandler: directiveHandler,

		timer: newTimer(),

		intercepting:     true,
		interceptingLock: new(sync.Mutex),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/message", c.handleMessage)
	mux.HandleFunc("/directive", c.handleDirective)
	mux.HandleFunc("/timeout", c.handleTimeout)

	c.server = &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}
	return c
}

func (c *ClientController) OutChan() chan *Message {
	return c.toNode
}

func (c *ClientController) SetReady() {
	c.readyLock.Lock()
	c.ready = true
	c.readyLock.Unlock()

	go c.sendMasterMessage(&masterRequest{
		Type: "RegisterPeer",
		Peer: &peer{
			ID:    c.peerID,
			Info:  c.peerInfo,
			Addr:  c.listenAddr,
			Ready: true,
		},
	})
}

func (c *ClientController) UnsetReady() {
	c.readyLock.Lock()
	c.ready = false
	c.readyLock.Unlock()

	go c.sendMasterMessage(&masterRequest{
		Type: "RegisterPeer",
		Peer: &peer{
			ID:    c.peerID,
			Info:  c.peerInfo,
			Addr:  c.listenAddr,
			Ready: false,
		},
	})
}

func (c *ClientController) IsReady() bool {
	c.readyLock.Lock()
	defer c.readyLock.Unlock()

	return c.ready
}

func (c *ClientController) SendMessage(to PeerID, msg string, intercept bool) error {

	select {
	case <-c.stopCh:
		return errors.New("controller stopped. EOF")
	default:
	}

	if !c.canIntercept() {
		return nil
	}

	c.fromNode <- &Message{
		From:      c.peerID,
		To:        to,
		ID:        "",
		Msg:       msg,
		Intercept: intercept,
	}
	return nil
}

func (c *ClientController) Start() error {
	errCh := make(chan error, 1)

	c.sendMasterMessage(&masterRequest{
		Type: "RegisterPeer",
		Peer: &peer{
			ID:    c.peerID,
			Info:  c.peerInfo,
			Addr:  c.listenAddr,
			Ready: false,
		},
	})

	go func() {
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	go c.poll()

	select {
	case e := <-errCh:
		return e
	case <-time.After(1 * time.Second):
		return nil
	}
}

func (c *ClientController) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer func() {
		cancel()
	}()

	close(c.stopCh)
	if err := c.server.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}

func (c *ClientController) poll() {
	for {
		select {
		case msg := <-c.fromNode:
			go c.sendMasterMessage(&masterRequest{
				Type:    "InterceptedMessage",
				Message: msg,
			})
		case <-c.stopCh:
			return
		}
	}
}

type peer struct {
	ID    PeerID                 `json:"id"`
	Addr  string                 `json:"addr"`
	Info  map[string]interface{} `json:"info,omitempty"`
	Ready bool                   `json:"ready"`
}

type masterRequest struct {
	Type    string   `json:"type"`
	Peer    *peer    `json:"peer,omitempty"`
	Message *Message `json:"message,omitempty"`
	Timeout *timeout `json:"timeout,omitempty"`
}

func (c *ClientController) sendMasterMessage(msg *masterRequest) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, "http://"+c.masterAddr, bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("response was not ok	")
	}
	return nil
}

func (c *ClientController) pause() {
	c.interceptingLock.Lock()
	c.intercepting = false
	c.interceptingLock.Unlock()
}

func (c *ClientController) resume() {
	c.interceptingLock.Lock()
	c.intercepting = true
	c.interceptingLock.Unlock()
}

func (c *ClientController) canIntercept() bool {
	c.interceptingLock.Lock()
	defer c.interceptingLock.Unlock()

	return c.intercepting
}
