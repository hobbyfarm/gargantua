package shell

import (
	"github.com/gorilla/websocket"
	"io"
	"sync"
)

type WSWrapper struct {
	io.ReadWriteCloser
	ws   *websocket.Conn
	mode int
	lock sync.RWMutex
}

func NewWSWrapper(ws *websocket.Conn, mode int) *WSWrapper {
	if ws == nil {
		return nil
	}

	return &WSWrapper{
		ws:   ws,
		mode: mode,
	}
}

func (w *WSWrapper) Write(data []byte) (n int, err error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	n = len(data)
	err = w.ws.WriteMessage(w.mode, data)

	if err != nil {
		n = 0
	}

	return n, err
}

func (w *WSWrapper) Read(out []byte) (n int, err error) {
	var data []byte

	_, data, err = w.ws.ReadMessage()

	return copy(out, data), nil
}

func (w *WSWrapper) Close() error {
	return w.ws.Close()
}
