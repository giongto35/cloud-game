test-e2e:
	cd e2e/coturn && ./test.sh
lint:
	golangci-lint run
assert:
	bash .github/assert-contributors.sh
	bash .github/lint-disallowed-functions-in-library.sh
	bash .github/lint-commit-message.sh
test:
	@./go.test.sh
