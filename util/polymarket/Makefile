.DEFAULT_GOAL := sync-docs

.PHONY: sync-docs snapshot-public-read snapshot-public-read-dry
sync-docs:
	./sync-docs.sh

snapshot-public-read:
	go run ./tools/cmd-polymarket-public-read-snapshot

snapshot-public-read-dry:
	go run ./tools/cmd-polymarket-public-read-snapshot --dry-run
