/** ****************************************************************************************************************** **
	Main file for the server side of K8MQ
	A message que system designed for use within a kubernetes system.
	This server service should run as a single pod deployment, with a maxSurge of 0, 
	so it completely "goes away" before the clients re-connect to it.
	
** ****************************************************************************************************************** **/

package main

import (
	"github.com/NathanRThomas/k8mq/server/models"

	"github.com/jessevdk/go-flags"
	
	"fmt"
	"context"
	"log"
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

const serviceVersion = "0.0.1"
const serviceName = "K8MQ Server"

// final local options object for this executable
var opts struct {
	models.OPTS
	Port string `short:"p" long:"port" description:"Specifies the target port to run on"`
}

var cfg struct {
	models.CFG
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
	models.Stack
	running bool 

	que *models.Que 
}

// init function for the app 
func (this *app) init() (models.Callback, error) {
	// default logger for formatted error messages
	this.ErrorLog = log.New (os.Stderr, "ERROR\t", log.LstdFlags | log.Lmicroseconds | log.Llongfile | log.LUTC)

	this.que = models.NewQue()
	
	return func() error {
		// close these in order
		return this.que.Close(time.Second * 20)

	}, nil // return any error from above
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
func (this *app) createServer (port string, wg *sync.WaitGroup, handler http.Handler) *http.Server {
	svr := &http.Server {
		Addr: ":" + port,
		ErrorLog: this.ErrorLog,
		Handler: handler, 
		ReadTimeout: time.Second * 30,
	}

	// listen for system interupt signals to quit the handler
	this.monitorForKill(func(){
		time.Sleep (time.Second * 5) // for being removed from the load balancer

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

	// parse the config
	err := models.ParseConfig(&cfg, opts.ConfigFile)
	if err != nil {
		log.Fatal(err) // bailing hard
	}

	opts.Info("Starting %s v%s", serviceName, serviceVersion)

	// check for a port being set
	if len(opts.Port) > 0 {
		cfg.Port = opts.Port // this wins
	} else if len(cfg.Port) == 0 { 
		cfg.Port = "8080"  // give it a default k8 is expecting
	}

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
			fmt.Println("TODO")

			err = finalDefer()
			if err != nil {
				app.ErrorLog.Printf("Final Defer Error\n%v\n", err)
			} // record our defered error
			os.Exit(0)

		default:
			finalDefer()
			log.Printf("Unknown command line argument '%s'\nCheckout our help for more info\n", arg)
			os.Exit(1)
		}
	}

	// we're good to keep going
	
	// create our server for listening for kubernetes health/live checks
	srv := app.createServer(cfg.Port, nil, app.routes())

	app.running = true // this app is now officially running

	log.Printf("%s v%s started on port %s\n", serviceName, serviceVersion, cfg.Port) // going to always record this starting message
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {            // Error starting or closing listener:
		log.Printf("Error %s ListenAndServe: %v", serviceName, err) // we want to know if this failed for another reason
	}

	opts.Info("exiting")

	app.StackTrace(finalDefer())
	
	os.Exit(0) //final exit
}
