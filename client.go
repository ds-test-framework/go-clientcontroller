package clientcontroller

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/ds-test-framework/scheduler/types"
)

var client *ReplicaClient = nil

var (
	ErrNoReplicaID        = errors.New("replica ID config should not be empty")
	ErrNoClientServerAddr = errors.New("client server address should not be empty")
	ErrNoImperiumAddr     = errors.New("imperium addr should not be empty")
)

type Config struct {
	ReplicaID        types.ReplicaID
	ImperiumAddr     string
	ClientServerAddr string
	ClientAdvAddr    string
	Info             map[string]interface{}
}

func validateConfig(c *Config) error {
	if c.ReplicaID == "" {
		return ErrNoReplicaID
	}
	if c.ImperiumAddr == "" {
		return ErrNoImperiumAddr
	}
	if c.ClientServerAddr == "" {
		return ErrNoClientServerAddr
	}
	if c.ClientAdvAddr == "" {
		c.ClientAdvAddr = c.ClientServerAddr
	}
	return nil
}

// Init initializes the global controller
func Init(
	config *Config,
	d DirectiveHandler,
	logger Logger,
) error {
	c, err := NewReplicaClient(config, d, logger)
	if err != nil {
		return err
	}
	client = c
	return nil
}

// GetController returns the global controller if initialized
func GetClient() (*ReplicaClient, error) {
	if client == nil {
		return nil, errors.New("controller not initialized")
	}
	return client, nil
}

// ReplicaClient should be used as a transport to send messages between replicas.
// It encapsulates the logic of sending the message to the `masternode` for further processing
// The ReplicaClient also listens for incoming messages from the master and directives to start, restart and stop
// the current replica.
//
// Additionally, the ReplicaClient also exposes functions to manage timers. This is key to our testing method.
// Timers are implemented as message sends and receives and again this is encapsulated from the library user
type ReplicaClient struct {
	directiveHandler DirectiveHandler
	config           *Config
	messageQ         *MessageQueue

	started     bool
	startedLock *sync.Mutex
	stopCh      chan bool

	timer     *timer
	ready     bool
	readyLock *sync.Mutex

	masterClient *http.Client
	server       *http.Server

	counter *Counter
	logger  Logger
}

// NewReplicaClient creates a ClientController
// It requires a DirectiveHandler which is used to perform directive actions such as start, stop and restart
func NewReplicaClient(
	config *Config,
	directiveHandler DirectiveHandler,
	logger Logger,
) (*ReplicaClient, error) {
	if logger == nil {
		logger = newDefaultLogger()
	}
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	c := &ReplicaClient{
		config:           config,
		messageQ:         NewMessageQueue(),
		stopCh:           make(chan bool),
		directiveHandler: directiveHandler,
		timer:            newTimer(),
		started:          false,
		startedLock:      new(sync.Mutex),
		ready:            false,
		readyLock:        new(sync.Mutex),
		counter:          NewCounter(),
		logger:           logger,

		masterClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:    10,
				MaxConnsPerHost: 1,
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/message",
		wrapHandler(c.handleMessage, postRequest),
	)
	mux.HandleFunc("/directive",
		wrapHandler(c.handleDirective, postRequest),
	)
	mux.HandleFunc("/timeout",
		wrapHandler(c.handleTimeout, postRequest),
	)
	mux.HandleFunc("/health", c.handleHealth)

	c.server = &http.Server{
		Addr:    config.ClientServerAddr,
		Handler: mux,
	}
	return c, nil
}

// Running returns true if the clientcontroller is running
func (c *ReplicaClient) IsRunning() bool {
	c.startedLock.Lock()
	defer c.startedLock.Unlock()
	return c.started
}

// SetReady sets the state of the replica to ready for testing
func (c *ReplicaClient) Ready() {
	c.readyLock.Lock()
	c.ready = true
	c.readyLock.Unlock()

	go c.sendMasterMessage(&masterRequest{
		Type: "RegisterReplica",
		Replica: &types.Replica{
			ID:    types.ReplicaID(c.config.ReplicaID),
			Info:  c.config.Info,
			Addr:  c.config.ClientAdvAddr,
			Ready: true,
		},
	})
}

// UnsetReady sets the state of the replica to not ready for testing
func (c *ReplicaClient) NotReady() {
	c.readyLock.Lock()
	c.ready = false
	c.readyLock.Unlock()

	go c.sendMasterMessage(&masterRequest{
		Type: "RegisterReplica",
		Replica: &types.Replica{
			ID:    types.ReplicaID(c.config.ReplicaID),
			Info:  c.config.Info,
			Addr:  c.config.ClientAdvAddr,
			Ready: false,
		},
	})
}

// IsReady returns true if the state is set to ready
func (c *ReplicaClient) IsReady() bool {
	c.readyLock.Lock()
	defer c.readyLock.Unlock()

	return c.ready
}

// Start will start the ClientController by spawning the polling goroutines and the server
// Start should be called before SetReady/UnsetReady
func (c *ReplicaClient) Start() error {
	if c.IsRunning() {
		return nil
	}
	errCh := make(chan error, 1)
	c.logger.Info("Starting client controller", "addr", c.config.ClientServerAddr)
	err := c.sendMasterMessage(&masterRequest{
		Type: "RegisterReplica",
		Replica: &types.Replica{
			ID:    c.config.ReplicaID,
			Info:  c.config.Info,
			Addr:  c.config.ClientAdvAddr,
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
func (c *ReplicaClient) Stop() error {
	if !c.IsRunning() {
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

func (c *ReplicaClient) unsetReady() {
	c.readyLock.Lock()
	defer c.readyLock.Unlock()
	c.ready = false
}
