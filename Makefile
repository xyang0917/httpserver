root:
	export ROOT=github.com/cncamp/golang;
.PHONY: root

release:
	echo "building httpserver binary"
	mkdir -p bin/arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/arm64 .

.PHONY: release

clean:
	rm -rf bin