all: build

build:
	go build -o crafty main.go server.go

run:
	go run main.go server.go server

clean:
	rm crafty
