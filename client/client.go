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
	"encoding/json"
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
	hashListeners map[string](chan *models.QueMessage)
	hashLocker sync.RWMutex 
}


//----- PRIVATE -----------------------------------------------------------------------------------------------------//

// when a message comes in, we want to 
func (this *Client) monitorMessages () {
	this.wgMessages.Add(1)
	defer this.wgMessages.Done()

	for msg := range this.messages {
		if msg == nil { break } // channel is closed

		if this.ctx.Err() != nil { break } // we're shutting down

		// write this out to our server
		// i'm pretty sure we'll be handling errors and reconnecting from the reading thread,
		// so as long as the conn isn't nil, assume this works
		ok := false 
		for i := 0; i < 4; i++ {
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

// handles monitoring the read channel as well as re-connecting to the main service when the connection is invalid
func (this *Client) read () {
	for {
		if this.conn == nil {
			this.opts.Warn("QUE: no connection to primary service : reconnecting")
			this.connect()
			continue 
		}

		// now that we have a connection that isn't nil 
		mType, data, err := this.conn.Read(this.ctx)
		if err == nil {
			this.opts.Info("RAW QUE: Found message to read : %s", string(data))

			if mType == websocket.MessageText && this.reader != nil {
				this.opts.Info("QUE: Found message to read : %s", string(data))

				// first see if we have an idHash with a receiver channel
				mHash := &models.MessageHashPrototype{}
				err = json.Unmarshal(data, mHash)
				if err == nil && len(mHash.IdHash) > 0 {
					// we got an id hash, let's see if we registered a listening channel
					this.hashLocker.Lock()

					if ch, ok := this.hashListeners[mHash.IdHash]; ok {
						ch <- &models.QueMessage{ Msg: data }
						close(ch) // now close this channel, we don't need it anymore
						delete(this.hashListeners, mHash.IdHash) // remove it from our map as well
						this.hashLocker.Unlock() // unlock the hash locker
						continue // don't do the regular reader
					}

					this.hashLocker.Unlock() // unlock the hash locker
				}

				if this.reader != nil { // in theory there may be a use where something only writes and never reads
					this.reader(data)
				}
			}
		} else if this.ctx.Err() == nil {
			this.opts.Warn("QUE: Read error : %v : reconnecting", err)
			this.connect()
		} else {
			break // we're done
		}
	}

	this.opts.Info("QUE: Read exited")
}

// handles connecting to the remote server
func (this *Client) connect () {
	
	// try this with a timeout
	for i := 0; i < 4; i++ {
		if this.ctx.Err() != nil { return } // bail, we're closing

		ctx, cancel := context.WithTimeout(this.ctx, time.Second * 3)
		defer cancel()

		conn, _, err := websocket.Dial (ctx, fmt.Sprintf("ws://%s:%d/que", this.serverUrl, this.port), nil)
		if err == nil {
			this.conn = conn // we're good, copy this over
			this.opts.Info("QUE: connected to %s:%d", this.serverUrl, this.port)
			return
		}
		
		time.Sleep(time.Second * time.Duration(int(math.Pow(2, float64(i))))) // sleep with a exp backoff
	}

	// this is bad, couldn't connect to the server
	this.opts.Warn("QUE: failed to connect to %s:%d", this.serverUrl, this.port)
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
		if this.conn != nil {
			this.conn.Close(websocket.StatusNormalClosure, "")
		}

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

// registers a one-time channel to pass the data to anytime the id hash matches
func (this *Client) RegisterOneTime (idHash string, ch chan *models.QueMessage) {
	if len(idHash) == 0 { return } // bail

	this.hashLocker.RLock() // lock it for reading
	this.hashListeners[idHash] = ch // set this locally
	this.hashLocker.RUnlock() // unlock it, we're done
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

	ret.hashListeners = make(map[string](chan *models.QueMessage))

	// using context to coordinate closing things
	ret.ctx, ret.ctxCancel = context.WithCancel(context.Background())

	go ret.monitorMessages() // monitor this channel as well
	go ret.read() // fire off the reader

	return ret, nil 
}
