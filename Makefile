build:
	tinygo build -target=pico -o bin/main.uf2 .

flash:
	tinygo flash -target=pico .

test:
	go test ./...
