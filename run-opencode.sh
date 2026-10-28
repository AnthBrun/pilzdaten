#!/usr/bin/env bash
# Opencode-Lauf #3: TASK3 (Piktogramm-Filter, Essbarkeit, bessere Bilder, Wald-Use-Case)
set -euo pipefail
cd /home/hermes/projects/pilzdaten
export PATH="$HOME/.opencode/bin:$HOME/.local/bin:$PATH"
export OPENAI_API_KEY="$(tr -d '[:space:]' < "$HOME/.config/opencode/kiconnect.apikey")"
exec opencode run "$(cat TASK3.md)" --model kiconnect/gpt-6-mitarbeitende --auto 2>&1 | tee opencode-run3.log
