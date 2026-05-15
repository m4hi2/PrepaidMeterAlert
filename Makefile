.PHONY: infra infra-down infra-logs mcreate serve %

mcreate:
	@test -n "$(filter-out $@,$(MAKECMDGOALS))" || (echo "usage: make mcreate <name>"; exit 1)
	go run . migrate create "$(filter-out $@,$(MAKECMDGOALS))"

%:
	@:

serve:
	go run . serve 2>&1 | tee -a /tmp/meterbot.log

migrate:
	go run . migrate up

mdown:
	go run . migrate down

infra:
	docker compose -f docker-compose.dev.yml up -d

infra-down:
	docker compose -f docker-compose.dev.yml down

infra-logs:
	docker compose -f docker-compose.dev.yml logs -f


