set shell := ["bash", "-cu"]

default: help

help:
	@just --list

bin:
	mkdir -p bin
	go build -o bin/jikascript ./cmd/jikascript

install:
	go install ./cmd/jikascript

clean:
	rm -rf build out
	@echo "Cleaned generated build outputs (build/, out/)"

[arg("board", long)]
[arg("port", long)]
erase board="ch32v003" port="": bin
	@echo "Erasing flash on board {{board}}"
	@minichlink -E {{ if port != "" { "-l " + port } else { "" } }}

[arg("board", long)]
[arg("port", long)]
unbrick board="ch32v003" port="": bin
	@echo "Unbricking board {{board}} (power-cycles target, resets option bytes, erases flash)"
	@minichlink -u {{ if port != "" { "-l " + port } else { "" } }}
