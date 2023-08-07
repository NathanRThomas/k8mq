/** ****************************************************************************************************************** **
	Client include for working with the K8MQ server

	Connects and handles retries for re-connecting
	Thread safe writing function as well as reading function with a function callback
	
** ****************************************************************************************************************** **/

package client 


import (
	"github.com/pkg/errors"
	"golang.org/x/net/websocket"
	
	"fmt"
	"context"
	"sync"
	"time"
	"log"
	"math"
	"io"
)

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- CONSTS ----------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

type readCallback = func([]byte)

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- STRUCTS ---------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

type queMessage struct {
	Msg []byte 
}

// main object
type Client struct {
	serverUrl string 
	port int 
	reader readCallback
	running bool 
	conn *websocket.Conn 	// The websocket connection.

	wg *sync.WaitGroup
	messages chan *queMessage
}


//----- PRIVATE -----------------------------------------------------------------------------------------------------//

// when a message comes in, we want to 
func (this *Client) monitorMessages () {
	this.wg.Add(1)
	defer this.wg.Done()

	for msg := range this.messages {
		if msg == nil { break } // channel is closed

		// write this out to our server
		// i'm pretty sure we'll be handling errors and reconnecting from the reading thread,
		// so as long as the conn isn't nil, assume this works
		ok := false 
		for i := 0; i < 5; i++ {
			if this.conn != nil {
				_, err := this.conn.Write(msg.Msg)
				if err == nil {
					ok = true 
					break 
				}
			}

			// if we're here, it's cause we couldn't send things, so try again
			time.Sleep(time.Second * time.Duration(int(math.Pow(2, float64(i))))) // sleep with a exp backoff
		}

		// not sure what to do, can't get into a thread lock by trying to write back to the channel
		// also we're clearly not connecting to the k8mq server
		// so just log it and move on
		if ok == false {
			log.Printf("couldn't write to the k8mq server : %s\n", string(msg.Msg))
		}
	}
}

func (this *Client) read () {
	this.wg.Add(1)
	defer this.wg.Done()

	for this.running {
		data, err := io.ReadAll (this.conn)
		if err == nil {
			this.reader(data)
		} else {
			log.Printf("read error : %v : reconnecting\n", err)
			this.connect()
		}
	}
}

// handles connecting to the remote server
func (this *Client) connect () error {
	var outErr error 
	// try this with a timeout
	for i := 0; i < 5; i++ {
		conn, err := websocket.Dial (fmt.Sprintf("ws://%s:%d/que", this.serverUrl, this.port), "", "http://k8mq-client")
		if err == nil {
			this.conn = conn // we're good, copy this over
			return nil 
		}
		outErr = err // use this for the return
		time.Sleep(time.Second * time.Duration(int(math.Pow(2, float64(i))))) // sleep with a exp backoff
	}

	// this is bad, couldn't connect to the server
	return errors.Wrap(outErr, "Remote K8MQ server did not respond")
}

// closes things and waits in its own thread
func (this *Client) closeAndWait (ch chan bool) {
	this.running = false // shut it down

	// close all the channels
	close(this.messages)

	this.wg.Wait() // wait for the threads to finish
	// they fininshed, so set the channel
	ch <- true 
}

//----- PUBLIC -----------------------------------------------------------------------------------------------------//

// defer function to close things down
func (this *Client) Close (tm time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), tm)
	defer cancel()

	done := make(chan bool)
	
	go this.closeAndWait (done)

	select {
	case <- done:
		// we finished normally and expectidly 
		this.conn.Close() // close this, we're done with it

	case <-ctx.Done():
		// this is bad, means the context timed out before things finished
		return errors.Errorf("Client timed out waiting for channels to close")
	}

	return nil // we're good
}

// adds a new message to go to our server connection
// this is thread safe
func (this *Client) NewMsg (msg []byte) {
	this.messages <- &queMessage {
		Msg: msg,
	}
}

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- PUBLIC FUNCTIONS ------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

// creates a new client object to connect, send and receive messages from our server
func NewClient (serverUrl string, port int, reader readCallback) (*Client, error) {
	if len(serverUrl) == 0 { return nil, errors.Errorf("remote K8MQ server url required, eg 'k8mq.default.svc'")}
	if port == 0 { port = 8080 } // default port

	ret := &Client{
		serverUrl: serverUrl,
		port: port,
		running: true,
		reader: reader,
	}

	// first step, let's try to connec to our server, if that doesn't work then we're done pretty quick
	err := ret.connect()
	if err != nil { return nil, err }
	
	ret.messages = make (chan *queMessage, 100) // this should be happening real quick, but there is a concern if the server is unreachable

	ret.wg = new(sync.WaitGroup)
	
	go ret.monitorMessages() // monitor this channel as well
	go ret.read() // fire off the reader

	return ret, nil 
}
