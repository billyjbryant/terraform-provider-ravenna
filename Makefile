.PHONY: build test lint docs fmt spec-diff

build:
	go build -o terraform-provider-ravenna .

test:
	go test ./... -count=1

lint:
	golangci-lint run ./...
	npx --yes markdownlint-cli2@0.23.2

fmt:
	go fmt ./...
	terraform fmt -recursive ./examples

docs:
	tfplugindocs generate --provider-name ravenna

spec-diff:
	./scripts/spec-diff.sh
