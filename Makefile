NAME=appmig

all: test build

test:
	go test -v ./...

build:
	go build -o $(NAME)

install:
	go install github.com/addsict/$(NAME)

clean:
	rm $(NAME)
