/** ****************************************************************************************************************** **
	Main file for the server side of K8MQ
	A message que system designed for use within a kubernetes system.
	This server service should run as a single pod deployment, with a maxSurge of 0, 
	so it completely "goes away" before the clients re-connect to it.

	This level is intended to be included in another single pod service. This is done in case one of the pods
	connected to the K8MQ server is "special" and need to be recieving all the messages all the time.

	If all the pods connecting to this service are the same you can run the main file located in the cmd folder
	
** ****************************************************************************************************************** **/

package server

import (
	"github.com/NathanRThomas/k8mq/models"

	"github.com/pkg/errors"
	
	"fmt"
	"context"
	"net/http"
	"sync"
	"time"
	"log/slog"
)

  //-------------------------------------------------------------------------------------------------------------------//
 //----- CONSTS ------------------------------------------------------------------------------------------------------//
//-------------------------------------------------------------------------------------------------------------------//

  //-------------------------------------------------------------------------------------------------------------------//
 //----- PRIVATE FUNCTIONS -------------------------------------------------------------------------------------------//
//-------------------------------------------------------------------------------------------------------------------//


  //-----------------------------------------------------------------------------------------------------------------------//
 //----- APP -------------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

type Server struct {
	opts models.OPTS // used for logging
	port int 
	reader models.ReadCallback
	
	svr *http.Server

	que *models.Que 
	wg *sync.WaitGroup
}

// actually handles the closing of things in a background process
func (this *Server) closeAndWait (ctx context.Context, done chan bool) {
	if this.svr != nil {
		// this shutsdown the server and returns once there's no more active connections.
		// but we should only have k8 connections anyway, so this should be pretty quick
		this.svr.Shutdown(ctx)
	}

	if this.que != nil {
		this.que.Close(time.Second * 20)
	}

	if this.wg != nil {
		this.wg.Wait() // wait for the server to shut down
	}

	done <- true // we're done
}

// designed to be run in its own go thread
func (this *Server) launchServer (port int) {
	// launch our server
	this.wg.Add(1)  // this gets 

	this.svr = &http.Server {
		Addr: fmt.Sprintf(":%d", port),
		Handler: this.routes(), 
		ReadTimeout: time.Second * 30,
	}

	slog.Info(fmt.Sprintf("K8MQ Server Started on port %d", port))

	if err := this.svr.ListenAndServe(); err != http.ErrServerClosed && err != nil {            // Error starting or closing listener:
		slog.Warn("K8MQ Server closed with an error : " + err.Error())
	}
	
	this.wg.Done() // we're done, the server isn't running anymore
}

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- PUBLIC FUNCTIONS ------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

// tries to close things down over this defined time period
func (this *Server) Close (tm time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), tm)
	defer cancel()

	done := make(chan bool)
	
	go this.closeAndWait (ctx, done)

	select {
	case <- done:
		// we finished normally and expectidly 
		
	case <-ctx.Done():
		// this is bad, means the context timed out before things finished
		return errors.Errorf("Server timed out waiting for channels to close")
	}

	return nil // we're good
}

// in case we want to fire out a new message to all connected listeners
func (this *Server) NewMsg (msg []byte) {
	if this.que != nil {
		this.que.NewMsg (msg) // repeat this to everyone
	}
}

// this should be fired as soon as k8 knows it's shutting down the k8mq service
func (this *Server) SendShutdown () {
	this.NewMsg ([]byte(models.ShutdownMessage))
	time.Sleep(time.Millisecond * 300) // give a little time to clients process this
}

// setting a reader changes the behavior so instead of re-broadcasting each message it returns each message to the reader instead
func NewServer (port int, reader models.ReadCallback) (*Server, error) {
	if port == 0 { port = models.DefaultPort } // default port

	ret := &Server{
		wg: new(sync.WaitGroup),
		reader: reader, // could be null
	}

	ret.que = models.NewQue(&ret.opts)

	// launch our server
	go ret.launchServer (port)

	return ret, nil // we're good
}
