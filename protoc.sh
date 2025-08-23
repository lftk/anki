#!/bin/sh

protoc --go_out=. --go_opt=paths=source_relative ./pb/anki.proto
