TIMESTAMP = $(shell date +"%Y%m%d%H%M%S")
BRANCH = $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
COMMIT = $(shell git rev-parse HEAD)
VERSION = $(shell git describe --tags)
LDFLAGSPREFIX = github.com/AliyunContainerService/image-syncer/pkg/utils
LDFLAGS = "-X $(LDFLAGSPREFIX).Version=$(VERSION) -X $(LDFLAGSPREFIX).Branch=$(BRANCH) -X $(LDFLAGSPREFIX).Commit=$(COMMIT) -X $(LDFLAGSPREFIX).Timestamp=$(TIMESTAMP)"
.PHONY: cmd clean

cmd: $(wildcard ./pkg/client/*.go ./pkg/sync/*.go ./pkg/tools/*.go ./cmd/*.go ./*.go)
	go build -ldflags $(LDFLAGS) -o image-syncer ./main.go

clean: 
	rm image-syncer