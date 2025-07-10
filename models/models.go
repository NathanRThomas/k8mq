/** ****************************************************************************************************************** **
	General models and "stuff" for the server
	
** ****************************************************************************************************************** **/

package models

import (
	"fmt"
	"crypto/sha256"
	"time"
)

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- CONSTS ----------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

const DefaultPort		= 8088
const ShutdownMessage	= "SHUTTING IT DOWN"

type Callback = func() error // generic callback function that returns an error

type ReadCallback = func([]byte) // reader interface for getting newly received messages

// I don't like having to check for a nil callback function so i created this 
func EmptyCallback () error {
	return nil 
}

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- STRUCTS ---------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

// This is designed to be inhereted by any system making message calls
// it allows us to register listeners based on the IdHash we receive
type MessageHashPrototype struct {
	IdHash string // unique string in case the caller wants to receive an ack/nack
	Body []byte	// added it here as we always need bytes in the body
}

// generates a random hash for us
func (this *MessageHashPrototype) SetIdHash () {
	h := sha256.New()
    h.Write([]byte(fmt.Sprintf("local-salt:%d:%s", time.Now().UnixNano(), this.Body)))
    this.IdHash = fmt.Sprintf("%x", h.Sum(nil))
}
