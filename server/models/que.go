/** ****************************************************************************************************************** **
	que list object for managing open connections and sending messages
	
** ****************************************************************************************************************** **/

package models

import (
	"github.com/pkg/errors"
	"github.com/gorilla/websocket"
	
	"context"
	"sync"
	"time"
	"log"
)

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- CONSTS ----------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- STRUCTS ---------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

type queConn struct {
	client *websocket.Conn
	ctx context.Context // to check if it's still good
}

type QueMessage struct {
	Msg []byte 
}

type Que struct {
	list []*queConn
	wg *sync.WaitGroup
	inConnection chan *queConn
	messages chan *QueMessage
}


//----- PRIVATE -----------------------------------------------------------------------------------------------------//

// adds connections to our list
func (this *Que) monitorIn () {
	this.wg.Add(1)
	defer this.wg.Done()

	for conn := range this.inConnection {
		if conn == nil { break } // channel is closed

		// add it to our list
		this.list = append (this.list, conn)
	}
}

// when a message comes in, we want to 
func (this *Que) monitorMessages () {
	this.wg.Add(1)
	defer this.wg.Done()

	for msg := range this.messages {
		if msg == nil { break } // channel is closed

		// writing to a bad connection is all i have, so i'm assuming things will be going away a lot
		// so let's re-create the list we need each time
		newList := make([]*queConn, 0, len(this.list))

		// we now need to send this message to all connected services
		for _, conn := range this.list {
			if conn.ctx.Err() == nil {
				err := conn.client.WriteMessage (1, msg.Msg) // write it out
				if err == nil {
					newList = append (newList, conn) // this one is still working, so keep it

				} else {
					// going to record these for now
					log.Println("client write failed, removing from que list")
				}
			} // else the context is gone, so don't include it anymore
		}

		this.list = newList // copy this over
	}
}

// closes things and waits in its own thread
func (this *Que) closeAndWait (ch chan bool) {
	// close all the channels
	close(this.inConnection)
	close(this.messages)

	this.wg.Wait() // wait for the threads to finish
	// they fininshed, so set the channel
	ch <- true 
}

//----- PUBLIC -----------------------------------------------------------------------------------------------------//

// defer function to close things down
func (this *Que) Close (tm time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), tm)
	defer cancel()

	done := make(chan bool)

	go this.closeAndWait (done)

	select {
	case <- done:
		// we finished normally and expectidly 
		
	case <-ctx.Done():
		// this is bad, means the context timed out before things finished
		return errors.Errorf("Que timed out waiting for channels to close")
	}

	return nil // we're good
}

// adds to our channel to add a connection, uses context as a test to make sure the connection is expected to be open
// this is thread safe
func (this *Que) AddConnection (ctx context.Context, c *websocket.Conn) {
	this.inConnection <- &queConn {
		client: c,
		ctx: ctx,
	}
}

// adds a new message to go to all connections
// this is thread safe
func (this *Que) NewMsg (msg []byte) {
	this.messages <- &QueMessage {
		Msg: msg,
	}
}

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- PUBLIC FUNCTIONS ------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

// creates a new que object to monitor for incoming connections
// sending of messages to existing connections
// and closing connections that are no longer open
func NewQue () *Que {
	ret := &Que{}

	ret.list = make([]*queConn, 0, 10) // TODO set some capacity so things load faster?

	ret.inConnection = make (chan *queConn, 10) // this doesn't need to be large, these should be getting pulled off real quick
	ret.messages = make (chan *QueMessage, 10) // again this should be happening real quick

	ret.wg = new(sync.WaitGroup)

	go ret.monitorIn()  // monitor this channel
	go ret.monitorMessages() // monitor this channel as well

	return ret 
}