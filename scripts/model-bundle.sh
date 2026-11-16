#!/usr/bin/env bash
# Stellt MODEL_DIR für die Foto-Vorschläge zusammen (siehe README, "Foto-Vorschläge").
#
#   scripts/model-bundle.sh <pilz-ki-Export> [MODEL_DIR]
#   scripts/model-bundle.sh ../pilz-ki/work/export/ft-fungi-effv2b3-final data/model
#
# Kopiert meta.json und model.int8.onnx aus dem Export von pilz-ki, lädt
# onnxruntime-web (Version und Prüfsumme fest eingetragen) und legt
# gzip-Varianten daneben, die der Server an passende Browser ausliefert.
set -euo pipefail

ORT_VERSION="1.31.1"
ORT_INTEGRITY="sha512-5GeH+bWQ+HxM10z7erGlsWqryrwMbSvm/1tkIsKTT48g0fI19VXVVC435t4t7VsPdSQeS7NmuZPXrQ73k82URw=="
ORT_FILES=(ort.wasm.min.mjs ort-wasm-simd-threaded.mjs ort-wasm-simd-threaded.wasm)

export_dir="${1:?Aufruf: $0 <pilz-ki-Export> [MODEL_DIR]}"
model_dir="${2:-data/model}"
for file in meta.json model.int8.onnx; do
  [[ -f "$export_dir/$file" ]] || { echo "$export_dir/$file fehlt" >&2; exit 1; }
done
grep -q '"fungus_prob"' "$export_dir/meta.json" || echo "Warnung: Modell ohne Gate (pilzki train-gate) – es schlägt auch auf Fotos ohne Pilz Merkmale vor." >&2

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
curl -fsSL -o "$tmp/ort.tgz" "https://registry.npmjs.org/onnxruntime-web/-/onnxruntime-web-${ORT_VERSION}.tgz"
actual="sha512-$(openssl dgst -sha512 -binary "$tmp/ort.tgz" | base64 | tr -d '\n')"
[[ "$actual" == "$ORT_INTEGRITY" ]] || { echo "Prüfsumme von onnxruntime-web stimmt nicht" >&2; exit 1; }
tar -xzf "$tmp/ort.tgz" -C "$tmp" "${ORT_FILES[@]/#/package/dist/}"

mkdir -p "$model_dir"
rm -rf "$model_dir"/ort-* "$model_dir"/*.gz
mkdir -p "$model_dir/ort-$ORT_VERSION"
for file in "${ORT_FILES[@]}"; do cp "$tmp/package/dist/$file" "$model_dir/ort-$ORT_VERSION/"; done
cp "$export_dir/meta.json" "$export_dir/model.int8.onnx" "$model_dir/"
find "$model_dir" -type f \( -name '*.json' -o -name '*.mjs' -o -name '*.wasm' -o -name '*.onnx' \) -exec gzip -kf9 {} \;

version="$(sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' "$model_dir/meta.json" | head -1)"
echo "MODEL_DIR $model_dir: Modell $version, onnxruntime-web $ORT_VERSION, $(du -sh "$model_dir" | cut -f1)"
