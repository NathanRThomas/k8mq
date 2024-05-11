/** ****************************************************************************************************************** **
	Client include for working with the K8MQ server

	Connects and handles retries for re-connecting
	Thread safe writing function as well as reading function with a function callback
	
** ****************************************************************************************************************** **/

package client 


import (
	"github.com/pkg/errors"
	"nhooyr.io/websocket"

	"github.com/NathanRThomas/k8mq/models"
	
	"fmt"
	"context"
	"sync"
	"time"
	"math"
)

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- CONSTS ----------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//



  //-----------------------------------------------------------------------------------------------------------------------//
 //----- STRUCTS ---------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

// main object
type Client struct {
	opts models.OPTS // used for logging
	serverUrl string 
	port int 
	reader models.ReadCallback
	ctx context.Context 
	ctxCancel context.CancelFunc
	conn *websocket.Conn 	// The websocket connection.

	wgMessages *sync.WaitGroup
	messages chan *models.QueMessage
}


//----- PRIVATE -----------------------------------------------------------------------------------------------------//

// when a message comes in, we want to 
func (this *Client) monitorMessages () {
	this.wgMessages.Add(1)
	defer this.wgMessages.Done()

	for msg := range this.messages {
		if msg == nil { break } // channel is closed

		// write this out to our server
		// i'm pretty sure we'll be handling errors and reconnecting from the reading thread,
		// so as long as the conn isn't nil, assume this works
		ok := false 
		for i := 0; i < 5; i++ {
			if this.conn != nil {
				err := this.conn.Write(this.ctx, websocket.MessageText, msg.Msg)
				if err == nil {
					ok = true 
					break 
				}
			}

			// if we're here, it's cause we couldn't send things, so try again
			this.opts.Warn("QUE: Unable to write message, sleeping")
			time.Sleep(time.Second * time.Duration(int(math.Pow(2, float64(i))))) // sleep with a exp backoff
		}

		// not sure what to do, can't get into a thread lock by trying to write back to the channel
		// also we're clearly not connecting to the k8mq server
		// so just log it and move on
		if ok == false {
			this.opts.Warn("QUE: Failed to write to the k8mq server : re-quing : %s", string(msg.Msg))
			this.messages <- msg
		}
	}

	this.opts.Info("QUE: Monitor exited")
}

func (this *Client) read () {
	for {
		mType, data, err := this.conn.Read(this.ctx)
		if err == nil {
			if mType == websocket.MessageText && this.reader != nil {
				this.opts.Info("QUE: Found message to read : %s", string(data))
				if this.reader != nil { // in theory there may be a use where something only writes and never reads
					this.reader(data)
				}
			}
		} else if this.ctx.Err() == nil {
			this.opts.Warn("QUE: Read error : %v : reconnecting\n", err)
			this.connect()
		} else {
			break // we're done
		}
	}

	this.opts.Info("QUE: Read exited")
}

// handles connecting to the remote server
func (this *Client) connect () error {
	var outErr error 
	// try this with a timeout
	for i := 0; i < 5; i++ {
		if this.ctx.Err() != nil { return errors.WithStack(this.ctx.Err()) } // bail, we're closing

		ctx, cancel := context.WithTimeout(this.ctx, time.Second * 3)
		defer cancel()

		conn, _, err := websocket.Dial (ctx, fmt.Sprintf("ws://%s:%d/que", this.serverUrl, this.port), nil)
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
	// close all the channels
	if this.messages != nil {
		close(this.messages)
	}

	if this.wgMessages != nil {
		this.wgMessages.Wait() // wait for the threads to finish
	}
	
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
		this.ctxCancel() // shut it down
		this.conn.Close(websocket.StatusNormalClosure, "")
		

	case <-ctx.Done():
		// this is bad, means the context timed out before things finished
		return errors.Errorf("Client timed out waiting for channels to close")
	}

	return nil // we're good
}

// adds a new message to go to our server connection
// this is thread safe
func (this *Client) NewMsg (msg []byte) {
	this.messages <- &models.QueMessage {
		Msg: msg,
	}
}

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- PUBLIC FUNCTIONS ------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

// creates a new client object to connect, send and receive messages from our server
func NewClient (serverUrl string, port int, reader models.ReadCallback, verbose []bool) (*Client, error) {
	if len(serverUrl) == 0 { return nil, errors.Errorf("remote K8MQ server url required, eg 'k8mq.default.svc'")}
	if port == 0 { port = models.DefaultPort } // default port

	ret := &Client{
		serverUrl: serverUrl,
		port: port,
		reader: reader,
		messages: make (chan *models.QueMessage, 100), // this should be happening real quick, but there is a concern if the server is unreachable
		wgMessages: new(sync.WaitGroup),
	}

	ret.opts.Verbose = verbose // so we can use the info and warn logging levels

	// using context to coordinate closing things
	ret.ctx, ret.ctxCancel = context.WithCancel(context.Background())

	// first step, let's try to connec to our server, if that doesn't work then we're done pretty quick
	err := ret.connect()
	if err != nil { return nil, err }
	
	go ret.monitorMessages() // monitor this channel as well
	go ret.read() // fire off the reader

	return ret, nil 
}
