all: demo

demo:
	go build -o client.exe examples/client/client.go
	GOOS=linux go build -o server.exe examples/server/server.go