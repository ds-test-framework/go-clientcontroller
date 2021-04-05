package clientcontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"
)

var c *ClientController = nil

// Init initialized the global controller
func Init(peer PeerID, masterAddr, listenAddr string, peerInfo map[string]interface{}, d DirectiveHandler) {
	c = NewClientController(peer, masterAddr, listenAddr, peerInfo, d)
}

// GetController returns the global controller if initialized
func GetController() (*ClientController, error) {
	if c == nil {
		return nil, errors.New("controller not initialized")
	}
	return c, nil
}

// PeerID identifier for a given Peer
type PeerID string

// Message that is to be sent between the replicas
type Message struct {
	Type      string `json:"type"`
	To        PeerID `json:"to"`
	From      PeerID `json:"from"`
	Msg       string `json:"msg"`
	ID        string `json:"id"`
	Intercept bool   `json:"intercept"`
}

// ClientController should be used as a transport to send messages between peers.
// It encapsulates the logic of sending the message to the `masternode` for further processing
// The ClientController also listens for incoming messages from the master and directives to start, restart and stop
// the current peer/replica.
//
// Additionally, the ClientController also exposes functions to manage timers. This is key to our testing method.
// Timers are implemented as message sends and receives and again this is encapsulated from the library user
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
	timer            *timer
	ready            bool
	readyLock        *sync.Mutex
	started          bool
	startedLock      *sync.Mutex
}

// NewClientController creates a ClientController
// It requires a DirectiveHandler which is used to perform directive actions such as start, stop and restart
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
		timer:            newTimer(),
		intercepting:     true,
		interceptingLock: new(sync.Mutex),
		started:          false,
		startedLock:      new(sync.Mutex),
		ready:            false,
		readyLock:        new(sync.Mutex),
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

// Running returns true if the clientcontroller is running
func (c *ClientController) Running() bool {
	c.startedLock.Lock()
	defer c.startedLock.Unlock()
	return c.started
}

// OutChan returns a channel which contains incoming messages for this replica/peer
func (c *ClientController) OutChan() chan *Message {
	return c.toNode
}

// SetReady sets the state of the replica to ready for testing
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

// UnsetReady sets the state of the replica to not ready for testing
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

// IsReady returns true if the state is set to ready
func (c *ClientController) IsReady() bool {
	c.readyLock.Lock()
	defer c.readyLock.Unlock()

	return c.ready
}

// SendMessage is to be used to send a message to another replica
// and can be marked as to be intercepted or not by the testing framework
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

// Start will start the ClientController by spawning the polling goroutines and the server
// Start should be called before SetReady/UnsetReady
func (c *ClientController) Start() error {
	if c.Running() {
		return nil
	}
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
		c.startedLock.Lock()
		c.started = true
		c.startedLock.Unlock()
		return nil
	}
}

// Stop will halt the clientcontroller and gracefully exit
func (c *ClientController) Stop() error {
	if !c.Running() {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer func() {
		cancel()
	}()

	close(c.stopCh)
	c.startedLock.Lock()
	c.started = false
	c.startedLock.Unlock()
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
