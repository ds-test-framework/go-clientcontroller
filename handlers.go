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
		c.resume()
		err = c.directiveHandler.Start()
	case stopAction:
		c.pause()
		err = c.directiveHandler.Stop()
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

type httpHandler func(http.ResponseWriter, *http.Request)

type httpErrorHandler func(http.ResponseWriter, *http.Request) error

func wrapHandler(handler httpHandler, validators ...httpErrorHandler) httpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, v := range validators {
			if err := v(w, r); err != nil {
				return
			}
		}
		handler(w, r)
	}
}

func postRequest(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		errS := fmt.Sprintf("method not allowed: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprint(w, errS)
		return errors.New(errS)
	}
	return nil
}
