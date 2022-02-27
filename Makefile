include .env
export $(shell sed 's/=.*//' .env)
NAMESPACE=budgeting
RECIPE=docker-compose.yaml

start: up run

run: build
	docker-compose -f ${RECIPE} -p budgeting up -d --force-recreate api

up: network instance migrate

network:
	-docker network create budgeting_network

build:
	docker build -t ${COMPONENT}_api .

instance:
	docker-compose -f ${RECIPE} -p budgeting up -d --force-recreate database
	docker-compose -f ${RECIPE} -p budgeting up -d --force-recreate redis
	sleep 5
migrate:
	@./scripts/migrate.sh

test:
	GIN_MODE=release go test ./... -cover -v -covermode count -coverprofile coverage.out 2>&1
	# GIN_MODE=release go test ./... -v -run '$UpdateUserReset' -covermode count -coverprofile coverage.out 2>&1
	go tool cover -func=coverage.out

exec_db:
	docker exec -ti budgeting_database_1 psql -U db -d testdb

exec_redis:
	docker exec -ti budgeting_redis_1 redis-cli -p 6379
