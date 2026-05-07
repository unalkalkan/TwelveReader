# Short-term Step 1: harden core Qwen3-TTS provider configuration

Created: 2026-05-06T18:49:55Z
Source: Telegram / human project owner

Owner wants to skip quality/production/medium-term work for now and focus only on short-term core system hardening through Step 4. Step 1 should keep Qwen3-TTS usage inside the current backend/provider pipeline, without vLLM-omni or workstation changes.

Implementation intent:
- Make the current OpenAI-compatible Qwen3-TTS provider path predictable and resilient.
- Keep provider discovery and voices stable for the app.
- Avoid workstation/vLLM-omni changes.
- Preserve persisted voice preview behavior.
