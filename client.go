package clientcontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/ds-test-framework/scheduler/types"
)

var c *ClientController = nil

type ReplicaID types.ReplicaID

type Message types.Message

// Init initialized the global controller
func Init(
	replicaID ReplicaID,
	masterAddr, listenAddr, externAddr string,
	replicaInfo map[string]interface{},
	d DirectiveHandler,
	logger Logger,
) {
	c = NewClientController(replicaID, masterAddr, listenAddr, externAddr, replicaInfo, d, logger)
}

// GetController returns the global controller if initialized
func GetController() (*ClientController, error) {
	if c == nil {
		return nil, errors.New("controller not initialized")
	}
	return c, nil
}

// ClientController should be used as a transport to send messages between replicas.
// It encapsulates the logic of sending the message to the `masternode` for further processing
// The ClientController also listens for incoming messages from the master and directives to start, restart and stop
// the current replica.
//
// Additionally, the ClientController also exposes functions to manage timers. This is key to our testing method.
// Timers are implemented as message sends and receives and again this is encapsulated from the library user
type ClientController struct {
	directiveHandler DirectiveHandler
	replicaID        ReplicaID

	masterAddr  string
	listenAddr  string
	externAddr  string
	replicaInfo map[string]interface{}

	server *http.Server

	fromNode chan *Message
	toNode   chan *Message

	resetting     bool
	resettingLock *sync.Mutex
	stopCh        chan bool

	intercepting     bool
	interceptingLock *sync.Mutex

	// timer *timer

	ready       bool
	readyLock   *sync.Mutex
	started     bool
	startedLock *sync.Mutex

	logger Logger
}

// NewClientController creates a ClientController
// It requires a DirectiveHandler which is used to perform directive actions such as start, stop and restart
func NewClientController(
	replicaID ReplicaID,
	masterAddr, listenAddr, externAddr string,
	replicaInfo map[string]interface{},
	directiveHandler DirectiveHandler,
	logger Logger,
) *ClientController {
	if logger == nil {
		logger = newDefaultLogger()
	}
	if externAddr == "" {
		externAddr = listenAddr
	}
	c := &ClientController{
		replicaID:        replicaID,
		masterAddr:       masterAddr,
		listenAddr:       listenAddr,
		externAddr:       externAddr,
		replicaInfo:      replicaInfo,
		fromNode:         make(chan *Message, 10),
		toNode:           make(chan *Message, 10),
		resetting:        false,
		resettingLock:    new(sync.Mutex),
		stopCh:           make(chan bool),
		directiveHandler: directiveHandler,
		// timer:            newTimer(),
		intercepting:     true,
		interceptingLock: new(sync.Mutex),
		started:          false,
		startedLock:      new(sync.Mutex),
		ready:            false,
		readyLock:        new(sync.Mutex),
		logger:           logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/message",
		wrapHandler(c.handleMessage, postRequest),
	)
	mux.HandleFunc("/directive",
		wrapHandler(c.handleDirective, postRequest),
	)
	// mux.HandleFunc("/timeout",
	// 	wrapHandler(c.handleTimeout, postRequest),
	// )
	mux.HandleFunc("/health", c.handleHealth)

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

// OutChan returns a channel which contains incoming messages for this replica
func (c *ClientController) OutChan() chan *Message {
	return c.toNode
}

// SetReady sets the state of the replica to ready for testing
func (c *ClientController) SetReady() {
	c.readyLock.Lock()
	c.ready = true
	c.readyLock.Unlock()

	go c.sendMasterMessage(&masterRequest{
		Type: "RegisterReplica",
		Replica: &types.Replica{
			ID:    types.ReplicaID(c.replicaID),
			Info:  c.replicaInfo,
			Addr:  c.externAddr,
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
		Type: "RegisterReplica",
		Replica: &types.Replica{
			ID:    types.ReplicaID(c.replicaID),
			Info:  c.replicaInfo,
			Addr:  c.externAddr,
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
func (c *ClientController) SendMessage(t string, to ReplicaID, msg []byte, intercept bool) error {

	select {
	case <-c.stopCh:
		return errors.New("controller stopped. EOF")
	default:
	}

	if !c.canIntercept() {
		return nil
	}

	c.fromNode <- &Message{
		Type:      t,
		From:      types.ReplicaID(c.replicaID),
		To:        types.ReplicaID(to),
		ID:        "",
		Data:      msg,
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
	c.logger.Info("Starting client controller", "addr", c.listenAddr)
	err := c.sendMasterMessage(&masterRequest{
		Type: "RegisterReplica",
		Replica: &types.Replica{
			ID:    types.ReplicaID(c.replicaID),
			Info:  c.replicaInfo,
			Addr:  c.externAddr,
			Ready: false,
		},
	})

	if err != nil {
		c.logger.Info("Failed to register replica", "err", err)
	}

	go func() {
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	go c.poll()

	select {
	case e := <-errCh:
		return e
	// TODO: deal with this hack
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
	c.logger.Info("Stopping client controller")
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

type masterRequest struct {
	Type    string
	Replica *types.Replica
	Message *Message
	// Timeout *timeout
	Event *event
	Log   *types.ReplicaLog
}

func (c *ClientController) sendMasterMessage(msg *masterRequest) error {
	var b []byte
	var route string
	var err error
	switch msg.Type {
	case "RegisterReplica":
		b, err = json.Marshal(msg.Replica)
		route = "/replica"
	case "InterceptedMessage":
		b, err = json.Marshal(msg.Message)
		route = "/message"
	// case "TimeoutMessage":
	// 	b, err = json.Marshal(msg.Timeout)
	// 	route = "/timeout"
	case "StateUpdate":
		b, err = json.Marshal(msg.Event)
		route = "/state"
	case "Log":
		b, err = json.Marshal(msg.Log)
		route = "/log"
	}
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, "http://"+c.masterAddr+route, bytes.NewBuffer(b))
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
