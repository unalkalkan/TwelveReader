# Short-term Step 3: upload starts segmentation and synthesis immediately with default voice

Created: 2026-05-06T18:49:55Z
Source: Telegram / human project owner

Change onboarding/book processing so upload does not wait for the user to pick persona voices before synthesis starts.

UX/pipeline intent:
- When a book is uploaded, segmentation and synthesis should start right away using the account/default voice.
- Newly discovered personas should initially map to the default voice automatically.
- Users can still change persona-specific mappings later.
- Existing single-user assumption is acceptable until real accounts exist.
