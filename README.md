# go-replicaclient

[![API Reference](https://img.shields.io/badge/api-reference-blue.svg)](https://ds-test-framework.github.io/docs/framework/api.html)

Imperium client to be run on each replica. Uses the Imperium API to send/receive messages. Refer [here](https://ds-test-framework.github.io/) for Imperium documentation.

## Installation

Fetch and install library using `go get`,
```
go get github.com/imperiumproject/go-replicaclient
```

## Usage

Define a directive handler to handle `Directive` messages from Imperium. For example,

```go
type Replica struct {
    ...
}

func (n *Replica) Start() error {}

func (n *Replica) Stop() error {}

func (n *Replica) Restart() error {}
```

To initialize the client, pass the ID, directive handler and optionally a logger that the client can use to log debug messages.

```go
import (
    "github.com/imperiumproject/go-replicaclient"
    "github.com/imperiumproject/imperium/types"
)

func main() {

    replica := NewReplica()
    ...
    err := replicaclient.Init(&replicaclient.Config{
        ReplicaID: types.ReplicaID(Replica.ID),
        ImperiumAddr: "<imperium_addr>",
        ClientServerAddr: "localhost:7074",
    }, replica, logger)
    if err != nil {
        ...
    }
}
```

The library maintains a single instance of `ReplicaClient` which can be used to send/receive messages or publish events

```go
client, _ := replicaclient.GetClient()
client.SendMessage(type, to, data, true)
...
message, ok := client.ReceiveMessage()
...

client.PublishEvent("Commit", map[string]string{
    "param1": "val1",
})

```

## Configuration

`Init` accepts a `Config` instance as the first argument which is defined as,

```go
type Config struct {
    ReplicaID        types.ReplicaID
    ImperiumAddr     string
    ClientServerAddr string
    ClientAdvAddr    string
    Info             map[string]interface{}
}
```
