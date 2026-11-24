#!/usr/bin/env bash
# Build + assinatura Authenticode do DosPrintGo (cross-compile Linux -> Windows)
#
# Variáveis opcionais:
#   SIGN_PFX   caminho do certificado .pfx (padrão: certs/dosprintgo.pfx, auto-assinado se não existir)
#   SIGN_PASS  senha do .pfx (padrão: dosprintgo)
#
# Para evitar o aviso do SmartScreen/"Editor desconhecido" use um certificado
# de Code Signing emitido por uma CA (OV/EV) em SIGN_PFX.
set -euo pipefail
cd "$(dirname "$0")"

SIGN_PFX="${SIGN_PFX:-certs/dosprintgo.pfx}"
SIGN_PASS="${SIGN_PASS:-dosprintgo}"
TS_URL="http://timestamp.digicert.com"

# 1. Recursos (ícone, manifesto, versão)
x86_64-w64-mingw32-windres -F pe-x86-64 -O coff -i cmd/dosprintgo/dosprintgo.rc -o cmd/dosprintgo/rsrc_windows_amd64.syso
i686-w64-mingw32-windres   -F pe-i386   -O coff -i cmd/dosprintgo/dosprintgo.rc -o cmd/dosprintgo/rsrc_windows_386.syso

# 2. Certificado auto-assinado (apenas se não existir)
if [ ! -f "$SIGN_PFX" ]; then
  echo ">> Gerando certificado auto-assinado em $SIGN_PFX"
  mkdir -p "$(dirname "$SIGN_PFX")"
  openssl req -x509 -newkey rsa:3072 -sha256 -days 1825 -nodes \
    -keyout "$SIGN_PFX.key" -out "$SIGN_PFX.crt" \
    -subj "/C=BR/O=Adonai/CN=DosPrintGo" \
    -addext "keyUsage=critical,digitalSignature" \
    -addext "extendedKeyUsage=codeSigning"
  openssl pkcs12 -export -inkey "$SIGN_PFX.key" -in "$SIGN_PFX.crt" -out "$SIGN_PFX" -passout "pass:$SIGN_PASS"
fi

# 3. Compilação
LDFLAGS="-s -w"
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$LDFLAGS" -o dist/DosPrintGo.exe     ./cmd/dosprintgo
GOOS=windows GOARCH=386   go build -trimpath -ldflags "$LDFLAGS" -o dist/DosPrintGo_x86.exe ./cmd/dosprintgo

# 4. Assinatura
for exe in dist/DosPrintGo.exe dist/DosPrintGo_x86.exe; do
  osslsigncode sign -pkcs12 "$SIGN_PFX" -pass "$SIGN_PASS" \
    -n "DosPrintGo" -i "https://github.com/cslemes/DosPrintGo" \
    -h sha256 -ts "$TS_URL" \
    -in "$exe" -out "$exe.signed"
  mv "$exe.signed" "$exe"
  osslsigncode verify -in "$exe" 2>&1 | grep -E "Signature verification|Number of signers" || true
done

ls -la dist/
