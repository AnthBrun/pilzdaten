#!/usr/bin/env bash
# Opencode-Lauf #9: TASK9 (Pilz-Lab-Rework — Essbarkeit Chip, Boden-Standard, Büschel-Minis, Substrat näher)
set -euo pipefail
cd /home/hermes/projects/pilzdaten
export PATH="$HOME/.opencode/bin:$HOME/.local/bin:$PATH"
export OPENAI_API_KEY="$(tr -d '[:space:]' < "$HOME/.config/opencode/kiconnect.apikey")"
PROMPT="$(cat TASK9.md)"
exec opencode run "$PROMPT" --model kiconnect/gpt-6-mitarbeitende --auto 2>&1 | tee opencode-run9.log
