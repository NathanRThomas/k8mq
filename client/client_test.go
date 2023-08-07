
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

	client, err := NewClient ("localhost", 8080, clientReadCallback)
	if err != nil { t.Fatal(err) }

	client.NewMsg([]byte("Hello World"))

	time.Sleep(time.Second * 10)

	err = client.Close(time.Second)
	if err != nil { t.Fatal(err) }
}
