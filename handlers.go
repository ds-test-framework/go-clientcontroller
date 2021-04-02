package clientcontroller

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
	req := &directiveMessage{}
	err = json.Unmarshal(bodyB, req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Not OK!")
		return
	}
	switch req.Action {
	case startAction:
		break
	case stopAction:
		break
	case restartAction:
		c.pause()
		err = c.directiveHandler.Restart()
		c.resume()
	case isReadyAction:
		if !c.IsReady() {
			err = errors.New("replica not ready")
		}
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Not Ok!")
		return
	}
	fmt.Fprintf(w, "Ok")
}

func (c *ClientController) handleTimeout(w http.ResponseWriter, r *http.Request) {
	bodyB, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Not OK!")
		return
	}
	defer r.Body.Close()

	t := &timeout{}
	err = json.Unmarshal(bodyB, t)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Not OK!")
		return
	}

	c.timer.FireTimeout(t.Type)
	fmt.Fprintf(w, "Ok")
}
