/** ****************************************************************************************************************** **
	Websockets FTW!!!
	This handles the incoming websocket requests
	Trackes the list of them in an internal memory que
** ****************************************************************************************************************** **/

package server 

import (
	"github.com/gorilla/websocket"
	
	"net/http"
	"strings"
)


  //-------------------------------------------------------------------------------------------------------------------------//
 //----- WEBSOCKETS --------------------------------------------------------------------------------------------------------//
//-------------------------------------------------------------------------------------------------------------------------//

// handles a wss error and records it if it's "bad"
func (this *Server) wssErr (err error) {
	if strings.Contains(err.Error(), "websocket: close 1000") {
		return // we're cool with this one, normal close status
	}

	if strings.Contains(err.Error(), "websocket: close 1005") {
		return // we're cool with this one, expected when the client closes things
	}

	if strings.Contains(err.Error(), "websocket: close 1006") {
		return // we're also cool with this one, happens when the lb kicks an inactive connection out
	}

	if strings.Contains(err.Error(), "use of closed network connection") {
		return // happens when we close the client while the reader is still waiting for data
	}

	// this is probably bad, so record it
	this.opts.Warn("k8mq wss error %v", err)
}

// websocket entry point
func (this *Server) wssHandle (w http.ResponseWriter, r *http.Request) {
	ctx := r.Context() // pass to the connect to test context to see if it's bad?
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		this.opts.Warn("k8mq wss upgrade error %v", err)
		return
	}
	
	defer c.Close() // close it eventually

	// add this to our flow of users
	this.que.AddConnection (ctx, c)

	// listener
	for {
		mType, msg, err := c.ReadMessage() // try to read json, that's what's up
		if err != nil {
			this.wssErr(err) // bailing and close the socket, they were naughty
			break
		}
		
		this.opts.Info("received message : %d : %s", mType, string(msg))

		if mType != 1 { continue } // only passing along 1 types right now, utf8

		if this.reader != nil {
			this.reader (msg) // we have a specific reader, so do use that instead
		} else {
			this.que.NewMsg (msg) // repeat this to everyone
		}
	}
}
