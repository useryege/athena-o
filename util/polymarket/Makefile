.DEFAULT_GOAL := sync-docs

.PHONY: sync-docs snapshot-public-read snapshot-public-read-dry
sync-docs:
	./sync-docs.sh

snapshot-public-read:
	go run ./util/polymarket/cmd-polymarket-public-read-snapshot

snapshot-public-read-dry:
	go run ./util/polymarket/cmd-polymarket-public-read-snapshot --dry-run
