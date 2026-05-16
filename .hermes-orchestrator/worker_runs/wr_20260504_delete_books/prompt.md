You are an OpenCode builder worker for Hermes Orchestrator. Work in /workspace/TwelveReader only.

Use model opencode-go/deepseek-v4-pro. Implement backlog item blg_delete_books_feedback for feedback fb_20260504_001 "Delete Books".

Human feedback body:
"Three Dots" inside the book's read/player view (/player?bookId={book_id}) and at the Continue Listening is not doing anything. Make these open a native frontend dropdown. In this dropdown for now there should be the "Delete Book" option. Implement deleting a book under this button.

Acceptance criteria:
1. Backend supports DELETE /api/v1/books/{bookId}.
2. Repository/storage deletion removes the book directory/artifacts for local storage and is safe if book does not exist returns 404 at handler level.
3. Frontend API client exposes deleteBook(bookId), and React Query hooks can invalidate/refetch book lists/status/segments after delete.
4. Player top-bar more-horiz button opens a small native in-app dropdown/menu with Delete Book.
5. Continue Listening more-vert button opens a small native in-app dropdown/menu with Delete Book and does not trigger parent card navigation.
6. Delete asks for confirmation using Alert/confirm as appropriate, calls backend, pauses/clears current playback if deleting current book when possible, and navigates away from deleted player.
7. Preserve current design style; no branding/watermarks.
8. Run relevant checks: go test ./... and npm run build in web-client if available. If a check fails because of pre-existing environment/dependency issue, document exactly.

Important implementation hints:
- Backend files: internal/api/book_handler.go, internal/book/repository.go, internal/storage/adapter.go/local.go, cmd/server/main.go.
- Frontend files: web-client/src/api/client.ts, web-client/src/api/hooks.ts, web-client/app/player.tsx, web-client/app/(tabs)/index.tsx.
- Use the existing Colors/theme style.
- Avoid broad refactors.

Required output: edit files, run checks, and write a concise summary to .hermes-orchestrator/worker_runs/wr_20260504_delete_books/result.json with status, files_changed, tests, and notes. Do not commit; supervisor owns commit.
