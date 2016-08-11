all:
	go build -o ./bin/setaria ./src

clean:
	@rm -fr bin pkg
