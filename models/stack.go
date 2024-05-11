/** ****************************************************************************************************************** **
	Stack tracing errors. pulled this out of main for fun
	
** ****************************************************************************************************************** **/

package models

import (
	"github.com/pkg/errors"
	
	"fmt"
	"context"
	"log"
	"testing"
)

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- CONSTS ----------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

//----- Error Handling -----------------------------------------------------------------------------------------------//
type stackTracer interface {
	StackTrace() errors.StackTrace
}

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- STACK -----------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

type Stack struct {
	ErrorLog *log.Logger
}

func (this *Stack) StackTrace (err error) {
	if err == nil { return }
	if this.ErrorLog != nil {
		this.ErrorLog.Println (err)
	} else {
		log.Println (err)
	}

	for _, ln := range StackTraceToArray (err) {
		fmt.Println (ln) // print out each line
	}
}

// simple wrapper when we want to create a new error and stack trace in the same call
func (this *Stack) TraceErr (msg string, params ...interface{}) {
	this.StackTrace (errors.Errorf(msg, params...))
}

// Checks the context and records an error with the stack if it's bad/expired
func (this *Stack) CtxOk (ctx context.Context) bool {
	this.StackTrace (errors.WithStack (ctx.Err())) // record any error with a stack
	return ctx.Err() == nil // return if we're ok
}

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- PUBLIC FUNCTIONS ------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//


// converts an error into an array of strings that contain the stack lines and files
func StackTraceToArray (err error) (ret []string) {
	if err, ok := err.(stackTracer); ok {
		for _, fr := range err.StackTrace() {
			ret = append (ret, fmt.Sprintf ("%+v", fr)) 
			// https://github.com/pkg/errors/blob/5dd12d0cfe7f152f80558d591504ce685299311e/stack.go#L52
			// to reference the sprintf output of fr
		}
	}
	return 
}

// used during testing so we can see where the unexpected error is coming from 
func TestingStackTrace (t *testing.T, err error) {
	if err == nil { return }
	for _, ln := range StackTraceToArray (err) {
		t.Logf("%s\n", ln)
	}

	t.Fatal(err)
}
