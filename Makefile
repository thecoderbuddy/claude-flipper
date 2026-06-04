build:
	go build -o /usr/local/bin/flipper .

release:
	GITHUB_TOKEN=$(GITHUB_TOKEN) HOMEBREW_TAP_GITHUB_TOKEN=$(GITHUB_TOKEN) goreleaser release --clean
