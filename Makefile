default: build

build:
	go build ./...

test:
	go test ./... -count=1

testacc:
	TF_ACC=1 NEXSPENCE_URL=http://localhost:8081 NEXSPENCE_USERNAME=admin NEXSPENCE_PASSWORD=admin123 \
		go test ./internal/provider/ -v -count=1 -timeout 30m

stack-up:
	docker compose -f docker-compose.acc.yml up -d --wait

stack-down:
	docker compose -f docker-compose.acc.yml down -v

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run

docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest generate
