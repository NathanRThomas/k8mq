
package client 

import (
	
	//"github.com/stretchr/testify/assert"
	
	"testing"
	"time"
	"log"
)

func clientReadCallback (b []byte) {
	log.Printf("clientReadCallback: %s", string(b))
}


func TestClient1 (t *testing.T) {
	verbose := make([]bool, 2)
	verbose[0], verbose[1] = true, true 

	client, err := NewClient ("localhost", 8088, clientReadCallback, verbose)
	if err != nil { t.Fatal(err) }

	client.NewMsg([]byte("{\"type\": \"Hello World\"}"))

	time.Sleep(time.Second * 10)

	err = client.Close(time.Second)
	if err != nil { t.Fatal(err) }
}

// this one is designed to test the reconnecting to the server
// so start this without the server, and then start the server on the same machine
func TestClient2 (t *testing.T) {
	verbose := make([]bool, 2)
	verbose[0], verbose[1] = true, true 
	
	client, err := NewClient ("localhost", 8088, clientReadCallback, verbose)
	if err != nil { t.Fatal(err) }

	client.NewMsg([]byte("{\"type\": \"Hello World\"}"))

	time.Sleep(time.Second * 60)

	err = client.Close(time.Second)
	if err != nil { t.Fatal(err) }
}
