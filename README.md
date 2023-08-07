## K8MQ
Message que designed for for Kubernetes deployments

***
### Example
func clientReadCallback (b []byte) {
	log.Printf("clientReadCallback: %s", string(b))
}


func TestClient1 (t *testing.T) {

	client, err := NewClient ("localhost", 8080, clientReadCallback)
	if err != nil { t.Fatal(err) }

	client.NewMsg([]byte("Hello World"))

	time.Sleep(time.Second * 3)

	err = client.Close(time.Second)
	if err != nil { t.Fatal(err) }
}
