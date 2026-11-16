#!/bin/sh
# Stellt MODEL_DIR für die Foto-Vorschläge zusammen (siehe README, "Foto-Vorschläge").
#
#   scripts/model.sh fetch [MODEL_DIR]                    Modell laut model.lock aus dem Release von pilz-ki
#   scripts/model.sh local <pilz-ki-Export> [MODEL_DIR]   Modell aus einem lokalen Export (zum Testen)
#
# MODEL_DIR ist standardmäßig data/model. onnxruntime-web kommt in beiden
# Fällen in der Version aus model.lock und wird gegen ORT_SHA256 geprüft.
# Daneben entstehen .gz-Varianten, die der Server an passende Browser schickt.
# Läuft auch mit BusyBox (Docker-Build, Stage "model").
set -eu

here=$(cd "$(dirname "$0")/.." && pwd)
lock="${MODEL_LOCK:-$here/model.lock}"
[ -f "$lock" ] || { echo "$lock fehlt" >&2; exit 1; }
# shellcheck disable=SC1090
. "$lock"

download() {
  if command -v curl >/dev/null 2>&1; then curl -fsSL -o "$2" "$1"; else wget -q -O "$2" "$1"; fi
}
check() {
  echo "$1  $2" | sha256sum -c - >/dev/null 2>&1 || { echo "Prüfsumme stimmt nicht: $2" >&2; exit 1; }
}

mode="${1:-}"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
case "$mode" in
  fetch)
    out="${2:-data/model}"
    [ -n "${MODEL_URL:-}" ] || { echo "model.lock enthält kein Modell (pilzki release in pilz-ki)" >&2; exit 1; }
    download "$MODEL_URL/meta.json" "$tmp/meta.json"
    download "$MODEL_URL/model.int8.onnx" "$tmp/model.int8.onnx"
    check "$META_SHA256" "$tmp/meta.json"
    check "$ONNX_SHA256" "$tmp/model.int8.onnx"
    ;;
  local)
    export_dir="${2:?Aufruf: $0 local <pilz-ki-Export> [MODEL_DIR]}"
    out="${3:-data/model}"
    cp "$export_dir/meta.json" "$export_dir/model.int8.onnx" "$tmp/"
    ;;
  *)
    echo "Aufruf: $0 fetch [MODEL_DIR] | $0 local <pilz-ki-Export> [MODEL_DIR]" >&2
    exit 2
    ;;
esac
grep -q '"fungus_prob"' "$tmp/meta.json" || echo "Warnung: Modell ohne Gate – es schlägt auch auf Fotos ohne Pilz Merkmale vor." >&2

download "https://registry.npmjs.org/onnxruntime-web/-/onnxruntime-web-$ORT_VERSION.tgz" "$tmp/ort.tgz"
check "$ORT_SHA256" "$tmp/ort.tgz"
mkdir -p "$tmp/ort"
tar -xzf "$tmp/ort.tgz" -C "$tmp/ort"

rm -rf "$out"
mkdir -p "$out/ort-$ORT_VERSION"
for file in ort.wasm.min.mjs ort-wasm-simd-threaded.mjs ort-wasm-simd-threaded.wasm; do
  cp "$tmp/ort/package/dist/$file" "$out/ort-$ORT_VERSION/"
done
cp "$tmp/meta.json" "$tmp/model.int8.onnx" "$out/"
find "$out" -type f ! -name '*.gz' | while read -r file; do gzip -9 -c "$file" > "$file.gz"; done

version=$(sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' "$out/meta.json" | head -n 1)
echo "MODEL_DIR $out: Modell $version, onnxruntime-web $ORT_VERSION"
