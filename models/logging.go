/** ****************************************************************************************************************** **
	Specifically for logging errors
	
** ****************************************************************************************************************** **/

package models 

import (
	"github.com/pkg/errors"
	
	"fmt"
	"os"
	"context"
	"log/slog"
	"testing"
)

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- CONSTS ----------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- STRUCTS ---------------------------------------------------------------------------------------------------------//
//-----------------------------------------------------------------------------------------------------------------------//


//----- Error Handling -----------------------------------------------------------------------------------------------//
type stackTracer interface {
	StackTrace() errors.StackTrace
}

type Logger struct {
}

func (this *Logger) Init () {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions { Level: slog.LevelDebug }))
	slog.SetDefault(logger) // so we just directly slog from somewhere without this original logger
}


func (this *Logger) StackTrace (err error) {
	if err == nil { return }
	slog.Error (err.Error())

	slog.Debug("Stacktrace", slog.Any("stacktrace", StackTraceToArray (err)))
}

// simple wrapper when we want to create a new error and stack trace in the same call
func (this *Logger) TraceErr (msg string, params ...interface{}) {
	this.StackTrace (errors.Errorf(msg, params...))
}

// Checks the context and records an error with the stack if it's bad/expired
func (this *Logger) CtxOk (ctx context.Context) bool {
	this.StackTrace (errors.WithStack (ctx.Err())) // record any error with a stack
	return ctx.Err() == nil // return if we're ok
}

  //-----------------------------------------------------------------------------------------------------------------------//
 //----- FUNCTIONS -------------------------------------------------------------------------------------------------------//
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
