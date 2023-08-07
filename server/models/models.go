/** ****************************************************************************************************************** **
	General models and "stuff" for the server
	
** ****************************************************************************************************************** **/

package models

import (
	
)

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- CONSTS ----------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

type Callback = func() error // generic callback function that returns an error

// I don't like having to check for a nil callback function so i created this 
func EmptyCallback () error {
	return nil 
}
