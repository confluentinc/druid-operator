SONAR_AUTH_TOKEN_VAULT_KV := token
SONAR_HOST := http://sonarqube.dp.confluent.io:9000
SONAR_SCANNER_ARGS := -X -Dsonar.host.url=$(SONAR_HOST)

SONAR_PROJECT_NAME := $(SEMAPHORE_PROJECT_NAME)

ifeq ($(CI),true)
POST_TEST_TARGETS += sonar-scan
endif

ifeq ($(SEMAPHORE_GIT_PR_NUMBER),)
	SONAR_SCANNER_ARGS += -Dsonar.branch.name=$(SEMAPHORE_GIT_BRANCH)
else
	SONAR_SCANNER_ARGS += -Dsonar.pullrequest.key=$(SEMAPHORE_GIT_PR_NUMBER)
	SONAR_SCANNER_ARGS += -Dsonar.pullrequest.branch=$(SEMAPHORE_GIT_PR_BRANCH)
	SONAR_SCANNER_ARGS += -Dsonar.pullrequest.base=$(SEMAPHORE_GIT_BRANCH)
endif


# upload sonarqube data to sonarqube
.PHONY: sonar-scan
sonar-scan:
	@sonar-scanner $(SONAR_SCANNER_ARGS) -Dsonar.login=$(shell vault kv get -field $(SONAR_AUTH_TOKEN_VAULT_KV) "v1/ci/kv/sonarqube/semaphore") || true

.PHONY: sonarqube-gate-pip-deps
sonarqube-gate-pip-deps:
	pip3 show confluent-ci-tools > /dev/null || pip3 install -U confluent-ci-tools


.PHONY: sonar-gate
sonar-gate: sonarqube-gate-pip-deps
	@sonarqube-ci gate \
		$(SONAR_PROJECT_NAME) \
		--pr-id $(SEMAPHORE_GIT_PR_NUMBER) \
		--token $(shell vault kv get -field $(SONAR_AUTH_TOKEN_VAULT_KV) "v1/ci/kv/sonarqube/semaphore") \
		--host $(SONAR_HOST)
