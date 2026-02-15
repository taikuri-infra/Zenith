.PHONY: build test lint docker-build run clean generate

BINARY := zenith-operator
DOCKER_IMAGE := zenith-operator
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	GO111MODULE=on go build -ldflags "-s -w" -o bin/$(BINARY) ./cmd/operator

run: build
	./bin/$(BINARY)

test:
	GO111MODULE=on go test -v -race -count=1 ./...

test-coverage:
	GO111MODULE=on go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) .

generate:
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
	controller-gen crd rbac:roleName=zenith-operator webhook paths="./..." output:crd:artifacts:config=config/crd/bases

clean:
	rm -rf bin/ coverage.out coverage.html
