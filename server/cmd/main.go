/** ****************************************************************************************************************** **
	Creates a service, mostly as an example, that listens for connections and broadcasts messages to all listeners
	
** ****************************************************************************************************************** **/

package main

import (
	"github.com/NathanRThomas/k8mq/models"
	"github.com/NathanRThomas/k8mq/server"

	"github.com/jessevdk/go-flags"
	
	"fmt"
	"context"
	"log"
	"log/slog"
	"os"
	"strings"
	"os/signal"
	"syscall"
	"net/http"
	"sync"
	"time"
)

  //-------------------------------------------------------------------------------------------------------------------//
 //----- CONSTS ------------------------------------------------------------------------------------------------------//
//-------------------------------------------------------------------------------------------------------------------//

const serviceVersion = "0.2.0"
const serviceName = "K8MQ Server"

// final local options object for this executable
var opts struct {
	models.OPTS
	WSSPort int `long:"wssport" description:"Port you want to run the websocket service on on" default:"8088"`
}
  //-------------------------------------------------------------------------------------------------------------------//
 //----- PRIVATE FUNCTIONS -------------------------------------------------------------------------------------------//
//-------------------------------------------------------------------------------------------------------------------//

func showHelp() {
	fmt.Printf("***************************\n%s : Version %s\n\n", serviceName, serviceVersion)

	fmt.Printf("\n**************************\n")
}

// handles parsing command arguments as well as setting up our opts object
func parseCommandLineArgs() []string {
	// parse things
	args, err := flags.Parse(&opts)
	if err != nil {
		log.Fatal(err)
	}

	if opts.Help {
		showHelp()
		os.Exit(0)
	}

	// check any args
	for _, arg := range args {
		switch strings.ToLower(arg) {
		case "help":
			showHelp()
			os.Exit(0)

		case "version":
			fmt.Printf("%s\n", serviceVersion)
			os.Exit(0)
		}
	}

	return args // return any arguments we don't know what to do with... yet
}


  //-----------------------------------------------------------------------------------------------------------------------//
 //----- APP -------------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

type app struct {
	models.Logger
	running bool 

	server *server.Server
}

// init function for the app 
func (this *app) init() (models.Callback, error) {
	// default logger for formatted error messages
	this.Logger.Init()
	
	var err error
	this.server, err = server.NewServer (opts.WSSPort, nil)

	return func() error {
		// close these in order
		return this.server.Close(time.Second * 20)

	}, err // return any error from above
}

// monitors for a kill sigterm to set the running = false
// fires a custom function when it exits
func (this *app) monitorForKill(fn func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-c // this sits until something comes into the channel, eg the notify interupts from above
		this.running = false
		if fn != nil {
			fn() // any callback function, like a timeout or shutdown
		}
	}()
}

// create a default server handler based on our routes
func (this *app) createServer (port int, wg *sync.WaitGroup, handler http.Handler) *http.Server {
	svr := &http.Server {
		Addr: fmt.Sprintf(":%d", port),
		Handler: handler, 
		ReadTimeout: time.Second * 30,
	}

	// listen for system interupt signals to quit the handler
	this.monitorForKill(func(){
		// we also need to make sure any background threads have finished, so wait for them here
		// otherwise we shutdown the server so k8 can't tell if we're happy or not
		if wg != nil {
			wg.Wait()
		}

		svr.Shutdown (context.Background())
	})
	
	return svr
}

  //-------------------------------------------------------------------------------------------------------------------//
 //----- MAIN --------------------------------------------------------------------------------------------------------//
//-------------------------------------------------------------------------------------------------------------------//

func main() {
	// first step, parse the command line params
	args := parseCommandLineArgs()

	slog.Info("Starting " + serviceName + " v" + serviceVersion)

	// early check for flags
	for _, arg := range args {
		switch strings.ToLower(arg) {
			// TODO if we have any flags at this point
		}
	}

	// main app for everything
	app := &app{}

	finalDefer, err := app.init() // init our application
	if err != nil {
		app.StackTrace(err)
		app.StackTrace(finalDefer())
		log.Fatal(err) // this is also super bad
	}

	// see what they're trying to do here
	for _, arg := range args {
		switch strings.ToLower(arg) {
		case "todo":
			// fmt.Println("TODO")

			err = finalDefer()
			if err != nil {
				slog.Error("Final Defer Error: " + err.Error())
			} // record our defered error
			os.Exit(0)

		default:
			finalDefer()
			slog.Error("Unknown command line argument '" + arg + "'\nCheckout our help for more info")
			os.Exit(1)
		}
	}

	// we're good to keep going
	
	// create our server for listening for kubernetes health/live checks
	srv := app.createServer(opts.Port, nil, app.routes())

	app.running = true // this app is now officially running

	slog.Info(fmt.Sprintf("%s v%s started on port %d\n", serviceName, serviceVersion, opts.Port)) // going to always record this starting message
	if err := srv.ListenAndServe(); err != http.ErrServerClosed && err != nil {            // Error starting or closing listener:
		slog.Warn(fmt.Sprintf("Error %s ListenAndServe: %v", serviceName, err)) // we want to know if this failed for another reason
	}

	slog.Info("exiting")

	app.StackTrace(finalDefer())
	
	os.Exit(0) //final exit
}
