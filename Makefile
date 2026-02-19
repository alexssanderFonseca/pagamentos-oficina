.PHONY: up down run create-table

up:
	docker-compose up -d

down:
	docker-compose down

run:
	go run cmd/server/main.go

create-table:
	aws --endpoint-url=http://localhost:4566 dynamodb create-table \
		--table-name Payments \
		--attribute-definitions \
			AttributeName=id,AttributeType=S \
			AttributeName=external_reference,AttributeType=S \
		--key-schema \
			AttributeName=id,KeyType=HASH \
		--global-secondary-indexes \
			"[{\"IndexName\": \"ExternalReferenceIndex\",\"KeySchema\":[{\"AttributeName\":\"external_reference\",\"KeyType\":\"HASH\"}],\"Projection\":{\"ProjectionType\":\"ALL\"},\"ProvisionedThroughput\":{\"ReadCapacityUnits\":5,\"WriteCapacityUnits\":5}}]" \
		--provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
		--region us-east-1
