package shell

import (
	"bytes"
	"github.com/gorilla/websocket"
	"io"
)

type InputWrapper struct {
	ws *websocket.Conn
}

const patternLen = 5

// ignoredInputs are the strange input bytes that we look for and drop
var ignoredInputs = [][patternLen]byte{
	{27, 91, 62, 48, 59},
	{27, 80, 48, 43, 114},
}

func (this *InputWrapper) Read(out []byte) (n int, err error) {
	var data []byte
	_, data, err = this.ws.ReadMessage()
	if err != nil {
		return 0, io.EOF
	}

	if len(data) >= patternLen {
		for i := range ignoredInputs {
			pattern := ignoredInputs[i]
			if bytes.Equal(pattern[:], data[:patternLen]) {
				return 0, nil
			}
		}
	}
	return copy(out, data), nil
}
