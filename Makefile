APP_NAME := DosPrintGo
DIST_DIR := dist
CMD_DIR  := ./cmd/dosprintgo
RC_FILE  := cmd/dosprintgo/dosprintgo.rc

SIGN_PFX  ?= certs/dosprintgo.pfx
SIGN_PASS ?= dosprintgo
TS_URL    ?= http://timestamp.digicert.com

LDFLAGS := -s -w

export PATH := $(HOME)/.local/bin:$(PATH)

.PHONY: all build build-amd64 build-386 resources cert sign test clean help

all: resources cert build sign ## Executa o ciclo completo: recursos, certificado, compilação e assinatura

help: ## Exibe os comandos disponíveis
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'

resources: ## Compila os recursos Windows (.rc) para arquivos .syso (ícone, manifesto, versão)
	@echo ">> Compilando recursos Windows (.syso)..."
	x86_64-w64-mingw32-windres -F pe-x86-64 -O coff -i $(RC_FILE) -o cmd/dosprintgo/rsrc_windows_amd64.syso
	i686-w64-mingw32-windres -F pe-i386 -O coff -i $(RC_FILE) -o cmd/dosprintgo/rsrc_windows_386.syso

cert: ## Gera certificado autoassinado para Code Signing se não existir
	@if [ ! -f "$(SIGN_PFX)" ]; then \
		echo ">> Gerando certificado autoassinado em $(SIGN_PFX)..."; \
		mkdir -p "$$(dirname $(SIGN_PFX))"; \
		openssl req -x509 -newkey rsa:3072 -sha256 -days 1825 -nodes \
			-keyout "$(SIGN_PFX).key" -out "$(SIGN_PFX).crt" \
			-subj "/C=BR/O=Adonai/CN=DosPrintGo" \
			-addext "keyUsage=critical,digitalSignature" \
			-addext "extendedKeyUsage=codeSigning"; \
		openssl pkcs12 -export -inkey "$(SIGN_PFX).key" -in "$(SIGN_PFX).crt" -out "$(SIGN_PFX)" -passout "pass:$(SIGN_PASS)"; \
	fi

build: build-amd64 build-386 ## Compila binários para Windows (x64 e x86)

build-amd64: resources ## Compila binário Windows x64
	@echo ">> Compilando $(DIST_DIR)/$(APP_NAME).exe (amd64)..."
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP_NAME).exe $(CMD_DIR)

build-386: resources ## Compila binário Windows x86
	@echo ">> Compilando $(DIST_DIR)/$(APP_NAME)_x86.exe (386)..."
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=386 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP_NAME)_x86.exe $(CMD_DIR)

sign: cert ## Assina os binários com Authenticode e Timestamp
	@echo ">> Assinando executáveis com Authenticode..."
	@for exe in $(DIST_DIR)/$(APP_NAME).exe $(DIST_DIR)/$(APP_NAME)_x86.exe; do \
		if [ -f "$$exe" ]; then \
			echo "   Assinando $$exe..."; \
			osslsigncode sign -pkcs12 "$(SIGN_PFX)" -pass "$(SIGN_PASS)" \
				-n "DosPrintGo" -i "https://github.com/cslemes/DosPrintGo" \
				-h sha256 -ts "$(TS_URL)" \
				-in "$$exe" -out "$$exe.signed"; \
			mv "$$exe.signed" "$$exe"; \
			osslsigncode verify -in "$$exe" 2>&1 | grep -E "Signature verification|Number of signers" || true; \
		fi; \
	done
	@ls -la $(DIST_DIR)/

test: ## Executa todos os testes unitários
	@echo ">> Executando testes..."
	go test -v ./...

clean: ## Limpa arquivos da pasta dist
	@echo ">> Limpando arquivos compilados..."
	rm -rf $(DIST_DIR)
