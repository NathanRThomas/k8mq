/** ****************************************************************************************************************** **
	Config object info for our server side of things
	
** ****************************************************************************************************************** **/

package models

import (
	"log"
)

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- STRUCTS ---------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

// for parsing command line arguments
type OPTS struct {
	Help bool `short:"h" long:"help" description:"Shows help message"`
	ConfigFile string `long:"config" description:"Sets the name and location for the config file to use"`
	Verbose []bool `short:"v" long:"verbose" description:"Show verbose debug information -v max of -vv"`
	Port int `short:"p" long:"port" description:"Port you want to run the service on" default:"8080"`
}

// logs info level stuff to our log
// depends on the verbose settings
func (this *OPTS) Info (msg string, params ...interface{}) {
	if len(this.Verbose) >= 2 {
		log.Printf(msg, params...)
	}
}

// logs wanring level stuff to our log
// depends on the verbose settings
func (this *OPTS) Warn (msg string, params ...interface{}) {
	if len(this.Verbose) >= 1 {
		log.Printf(msg, params...)
	}
}

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- PUBLIC FUNCTIONS ------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

