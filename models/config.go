/** ****************************************************************************************************************** **
	Config object info for our server side of things
	
** ****************************************************************************************************************** **/

package models

import (

)

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- STRUCTS ---------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

// for parsing command line arguments
type OPTS struct {
	Help bool `short:"h" long:"help" description:"Shows help message"`
	Port int `short:"p" long:"port" description:"Port you want to run the service on" default:"8080"`
}

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- PUBLIC FUNCTIONS ------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

