/** ****************************************************************************************************************** **
	General models and "stuff" for the server
	
** ****************************************************************************************************************** **/

package models

import (
	
)

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- CONSTS ----------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

const DefaultPort		= 8088

type Callback = func() error // generic callback function that returns an error

type ReadCallback = func([]byte) // reader interface for getting newly received messages

// I don't like having to check for a nil callback function so i created this 
func EmptyCallback () error {
	return nil 
}

