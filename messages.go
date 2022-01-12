package clientcontroller

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/ds-test-framework/scheduler/types"
)

// MessageQueue datastructure to store the messages in a FIFO queue
type MessageQueue struct {
	messages []*types.Message
	lock     *sync.Mutex
	size     int
}

// NewMessageQueue returns an empty MessageQueue
func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		messages: make([]*types.Message, 0),
		size:     0,
		lock:     new(sync.Mutex),
	}
}

func (q *MessageQueue) Pop() (*types.Message, bool) {
	q.lock.Lock()
	defer q.lock.Unlock()
	if q.size < 1 {
		return nil, false
	}
	front := q.messages[0]
	q.size = q.size - 1
	q.messages = q.messages[1:]
	return front, true
}

// Add adds a message to the queue
func (q *MessageQueue) Add(m *types.Message) {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.messages = append(q.messages, m)
	q.size = q.size + 1
}

// Flush clears the queue of all messages
func (q *MessageQueue) Flush() {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.messages = make([]*types.Message, 0)
	q.size = 0
}

type masterRequest struct {
	Type    string
	Replica *types.Replica
	Message *types.Message
	// Timeout *timeout
	Event *event
	Log   *log
}

func (c *ReplicaClient) sendMasterMessage(msg *masterRequest) error {
	var b []byte
	var route string
	var err error
	switch msg.Type {
	case "Event":
		b, err = json.Marshal(msg.Event)
		route = "/event"
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
	addr := "http://" + c.config.ImperiumAddr + route
	req, err := http.NewRequest(http.MethodPost, addr, bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	client := c.masterClient
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

// SendMessage is to be used to send a message to another replica
// and can be marked as to be intercepted or not by the testing framework
func (c *ReplicaClient) SendMessage(t string, to types.ReplicaID, msg []byte, intercept bool) error {

	select {
	case <-c.stopCh:
		return errors.New("controller stopped. EOF")
	default:
	}

	if !c.IsReady() {
		return nil
	}
	from := c.config.ReplicaID
	message := &types.Message{
		Type:      t,
		From:      from,
		To:        to,
		ID:        c.counter.NextID(from, to),
		Data:      msg,
		Intercept: intercept,
	}
	err := c.sendMasterMessage(&masterRequest{
		Type:    "InterceptedMessage",
		Message: message,
	})
	if err != nil {
		return err
	}
	c.PublishEventAsync(MessageSendEventType, map[string]string{
		"message_id": message.ID,
	})
	return nil
}

func (c *ReplicaClient) ReceiveMessage() (*types.Message, bool) {
	return c.messageQ.Pop()
}
