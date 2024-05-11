## K8MQ
Message que designed for for Kubernetes deployments

***

### All pods the same
If all the pods are the same, and you just need them to broadcast messages to eachother.
You can run a simple service like the one in server/cmd.
This creates a service that runs on port 8080, by default, with an additional
websocket service running on port 8088.

### Special Pod
If you have 1 pod that's special, handles thread safe things, and you *always* 
need that pod to be running for things to work.
Then you can just import the server like cmd/server does.
You can also pass the NewServer function a reader function.
This changes the behavior so instead of re-broadcasting all messages to
all connected services.
All incoming messages only get read into that function.
