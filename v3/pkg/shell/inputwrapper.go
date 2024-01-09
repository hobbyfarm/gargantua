package shell

import (
	"bytes"
	"io"
	"strconv"

	"github.com/gorilla/websocket"
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

	// check if data is a resize event
	if SIGWINCH.MatchString(string(data)) {
		size := SIGWINCH.FindStringSubmatch(string(data))

		h, _ := strconv.Atoi(size[1])
		w, _ := strconv.Atoi(size[2])
		ResizePty(h, w)
		return 0, nil
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
