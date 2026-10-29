#!/usr/bin/env bash
# Opencode-Lauf #10: TASK10 (Kombis ohne Daten ausgrauen — /api/available-tags)
set -euo pipefail
cd /home/hermes/projects/pilzdaten
export PATH="$HOME/.opencode/bin:$HOME/.local/bin:$PATH"
export OPENAI_API_KEY="$(tr -d '[:space:]' < "$HOME/.config/opencode/kiconnect.apikey")"
PROMPT="$(cat TASK10.md)"
exec opencode run "$PROMPT" --model kiconnect/gpt-6-mitarbeitende --auto 2>&1 | tee opencode-run10.log
