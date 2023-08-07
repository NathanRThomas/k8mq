/** ****************************************************************************************************************** **
	Websockets FTW!!!
	This handles the incoming websocket requests
	Trackes the list of them in an internal memory que
** ****************************************************************************************************************** **/

package main 

import (
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	
	"net/http"
	"strings"
)


  //-------------------------------------------------------------------------------------------------------------------------//
 //----- WEBSOCKETS --------------------------------------------------------------------------------------------------------//
//-------------------------------------------------------------------------------------------------------------------------//

// handles a wss error and records it if it's "bad"
func (this *app) wssErr (err error) {
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
	this.StackTrace(errors.WithStack(err))
}

// websocket entry point
func (this *app) wssHandle (w http.ResponseWriter, r *http.Request) {
	ctx := r.Context() // pass to the connect to test context to see if it's bad?
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		this.StackTrace(err)
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
		
		opts.Info("received message : %d : %s", mType, string(msg))

		if mType != 1 { continue } // only passing along 1 types right now, utf8

		this.que.NewMsg (msg) // repeat this to everyone
	}
}
