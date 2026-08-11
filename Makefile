build:
	tinygo build -target=pico -o main.uf2 main.go

flash:
	tinygo flash -target=pico main.go
