.PHONY: cmd clean

cmd: $(wildcard ./pkg/client/*.go ./pkg/sync/*.go ./pkg/tools/*.go ./cmd/*.go ./*.go)
	CGO_ENABLED=0 go build -o image-syncer ./main.go

clean: 
	rm image-syncer
