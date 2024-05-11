/** ****************************************************************************************************************** **
	Restful endpoints used to do the kubernetes health/live checks
	
** ****************************************************************************************************************** **/

package main 

import (

	"github.com/justinas/alice"
	"github.com/gorilla/mux"
	
	"net/http"
	"encoding/json"
)

  //-------------------------------------------------------------------------------------------------------------------------//
 //----- MIDDLEWARE --------------------------------------------------------------------------------------------------------//
//-------------------------------------------------------------------------------------------------------------------------//

func (this *app) defaultGet (w http.ResponseWriter, r *http.Request) {
	out, _ := json.Marshal (serviceName + " " + serviceVersion)
	w.Write (out)
}

func (this *app) liveCheck (next http.Handler) http.Handler {
	return http.HandlerFunc (func(w http.ResponseWriter, r *http.Request) {
		if this.running {
			next.ServeHTTP(w, r)
		} else {
			//if we're here it's bad
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Not Running"))
		}
    })
}

func (this *app) readyCheck (next http.Handler) http.Handler {
	return http.HandlerFunc (func(w http.ResponseWriter, r *http.Request) {
		if this.running {
			next.ServeHTTP(w, r)
		} else {
			//if we're here it's bad
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Not Running"))
		}
    })
}

func (this *app) thingsLookGood (w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Things look good")) //we're good
}

  //-------------------------------------------------------------------------------------------------------------------------//
 //----- ENTRY POINTS ------------------------------------------------------------------------------------------------------//
//-------------------------------------------------------------------------------------------------------------------------//

func (this *app) routes () *mux.Router {
	mux := mux.NewRouter().StrictSlash(true)
	
	// just something easy to get
	mux.Handle ("/", alice.New().ThenFunc(this.defaultGet)).Methods (http.MethodGet, http.MethodOptions)

	liveCheck := alice.New (this.liveCheck)

	// kubernetes stuff
	mux.Handle("/status/live", liveCheck.ThenFunc(this.thingsLookGood)).Methods(http.MethodGet, http.MethodOptions)

	mux.Handle("/status/ready", liveCheck.Append(this.readyCheck).ThenFunc(this.thingsLookGood)).Methods(http.MethodGet, http.MethodOptions)
	return mux
}
