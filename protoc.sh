#!/bin/sh

protoc --go_out=. --go_opt=paths=source_relative ./internal/pb/anki.proto
