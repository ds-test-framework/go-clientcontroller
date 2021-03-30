package godriver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

func (c *ClientController) handleMessage(w http.ResponseWriter, r *http.Request) {
	bodyB, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Not OK!")
		return
	}
	defer r.Body.Close()
	req := &Message{}
	err = json.Unmarshal(bodyB, req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Not OK!")
		return
	}
	c.toNode <- req
	fmt.Fprintf(w, "Ok!")
}

func (c *ClientController) handleDirective(w http.ResponseWriter, r *http.Request) {
	bodyB, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Not OK!")
		return
	}
	defer r.Body.Close()
	req := &DirectiveMessage{}
	err = json.Unmarshal(bodyB, req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Not OK!")
		return
	}
	switch req.Action {
	case StartAction:
		break
	case StopAction:
		break
	case RestartAction:
		c.pause()
		err = c.directiveHandler.Restart()
		c.resume()
		break
	case IsReadyAction:
		if !c.IsReady() {
			err = errors.New("Replica not ready")
		}
		break
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Not Ok!")
		return
	}
	fmt.Fprintf(w, "Ok")
}
