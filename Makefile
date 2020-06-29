build:
	GOOS=windows CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -o youtube_uploader.exe main.go