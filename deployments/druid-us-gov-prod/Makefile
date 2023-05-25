.PHONY: ensure-aws-token
ensure-aws-token:	
	@if [[ -z "${AWS_SESSION_TOKEN}" ]] ; then \
	echo "please initialize AWS_SESSION_TOKEN"; \
	exit 1; \
	fi ;

.PHONY: helmfile-apply
helmfile-apply: ensure-aws-token
	helmfile --interactive apply

clone-operator-repo:
	cd ../.. && rm -rf druid-operator && make clone-druid-operator-repo
