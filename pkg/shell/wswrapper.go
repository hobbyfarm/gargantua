package shell

import (
	"io"
	"sync"
	"unicode/utf8"

	"github.com/gorilla/websocket"
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
	err = w.ws.WriteMessage(w.mode, validUtf8(data))

	if err != nil {
		n = 0
	}

	return n, err
}

func validUtf8(b []byte) []byte {
	if !utf8.Valid(b) {
		s := string(b)
		v := make([]rune, 0, len(s))
		for i, r := range s {
			if r == utf8.RuneError {
				_, size := utf8.DecodeRuneInString(s[i:])
				if size == 1 {
					continue
				}
			}
			v = append(v, r)
		}
		s = string(v)
		return []byte(s)
	}
	return b
}

func (w *WSWrapper) Read(out []byte) (n int, err error) {
	var data []byte

	_, data, err = w.ws.ReadMessage()

	return copy(out, data), nil
}

func (w *WSWrapper) Close() error {
	return w.ws.Close()
}
