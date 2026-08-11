build:
	tinygo build -target=pico -o bin/main.uf2 main.go

flash:
	tinygo flash -target=pico main.go

test:
	go test ./...
