#!/bin/bash
# Roda todos os testes do projeto
# Uso: ./test.sh          (todos)
#      ./test.sh Config    (filtrar por nome)
CGO_ENABLED=1 go test -tags fts5 -v -count=1 -timeout 60s ${1:+-run "$1"} ./...
