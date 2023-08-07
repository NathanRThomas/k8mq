/** ****************************************************************************************************************** **
	Config object info for our server side of things
	
** ****************************************************************************************************************** **/

package models

import (
	"github.com/pkg/errors"
	
	"log"
	"encoding/json"
	"os"
)

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- CONSTS ----------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- CFG -------------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

type CFG struct {
	Port string

}

// for parsing command line arguments
type OPTS struct {
	Help bool `short:"h" long:"help" description:"Shows help message"`
	ConfigFile string `long:"config" description:"Sets the name and location for the config file to use"`
	Verbose []bool `short:"v" long:"verbose" description:"Show verbose debug information -v max of -vv"`
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

func ParseConfig (cfg interface{}, cfgLoc string) error {
	if len(cfgLoc) == 0 {
		cfgLoc = os.Getenv("CONFIG") // default to the env variable

		if len(cfgLoc) == 0 { return nil } // they don't need a config file for this adventure
	}

	config, err := os.Open(cfgLoc)
	if err != nil { return errors.WithStack (err) }

	jsonParser := json.NewDecoder (config)
	return errors.WithStack(jsonParser.Decode (cfg))
}
