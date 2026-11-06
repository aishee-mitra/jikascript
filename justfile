set shell := ["bash", "-cu"]

default: help

help:
	@just --list

bin:
	go build -o jikascript ./cmd/jikascript

install:
	go install ./cmd/jikascript

check file="examples/blink.jika": bin
	./jikascript check {{file}}

transpile file="examples/blink.jika" out="build/blink.c": bin
	mkdir -p "$(dirname "{{out}}")"
	./jikascript transpile {{file}} -o {{out}}

build source="examples/blink.jika" optimize="size": bin
	if [ -z "${JIKASCRIPT_CH32FUN:-}" ]; then echo "JIKASCRIPT_CH32FUN must point to a CH32fun checkout"; exit 1; fi
	./jikascript build {{source}} --target ch32v003 --optimize {{optimize}}

build-c source="build/blink.c" optimize="size": bin
	if [ -z "${JIKASCRIPT_CH32FUN:-}" ]; then echo "JIKASCRIPT_CH32FUN must point to a CH32fun checkout"; exit 1; fi
	./jikascript build {{source}} --target ch32v003 --optimize {{optimize}}
