gen-env-docs:
	@echo '| Description | Environment variable |'
	@echo '|-------------|----------------------|'
	@go run main.go --help 2>&1 | grep '\$$' | sed -r 's|^.*?\-\-[a-zA-Z0-9=.-]+[ ]+|\||g' | tr '[$$' '|`' | sed 's/]/`|/g'

.PHONY: gen-env-docs