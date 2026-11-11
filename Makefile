.PHONY: all build build-web build-go test run serve clean

all: build

build-web:
	cd web && npm install && npm run build

build-go:
	go build -v -o agentloop ./cmd/agentloop

build: build-web build-go

test:
	go test -v ./...
	cd web && npm run build

run:
	./agentloop run --file ./examples/invoice_reconciliation.yaml

serve:
	./agentloop serve --port 8080 --db ./agentloop.db

clean:
	rm -f agentloop agentloop.exe *.db *.db-wal *.db-shm
