.PHONY: help
help:	## Show target helps
	@echo "set ENV variable and call targets:"
	@echo
	@grep -E '^[a-zA-Z_%-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\t\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: init
init:	## run terraform init
	docker compose run --rm ${COMPOSE_OPTS} opentofu init

.PHONY: lint
lint:	## lint terraform files
	docker compose run --rm ${COMPOSE_OPTS} opentofu validate
	docker compose run --rm tflint --recursive
	docker compose run --rm ${COMPOSE_OPTS} opentofu fmt -recursive -check -diff .

.PHONY: format
format:	## format terraform files
	docker compose run --rm ${COMPOSE_OPTS} opentofu fmt -recursive .

.PHONY: lock
lock:	## create/update .terraform.lock.hcl file
	docker compose run --rm ${COMPOSE_OPTS} opentofu providers lock -platform=linux_amd64 -platform=linux_arm64

.PHONY: plan
plan:	## run terraform plan
	docker compose run --rm ${COMPOSE_OPTS} opentofu plan

.PHONY: apply
apply:	## run terraform apply
	docker compose run --rm ${COMPOSE_OPTS} opentofu apply

.PHONY: destroy
destroy:	## run terraform destroy
	docker compose run --rm terraform destroy
