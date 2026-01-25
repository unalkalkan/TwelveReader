# Twelve Reader - Web Client MVP

This is the Web Client MVP for Twelve Reader, built with React + TypeScript + Tamagui + TanStack Query + Zod.

## Features

- **Book Upload**: Upload TXT, PDF, or ePUB books with metadata (title, author, language)
- **Processing Status**: Real-time monitoring of book processing (parsing, segmentation, synthesis)
- **Audio Playback**: Play synthesized audio segments with synchronized text display
- **Voice Management**: View speaker assignments and voice descriptions
- **Clean UI**: Modern, responsive interface built with Tamagui

## Tech Stack

- **React 19**: UI framework
- **TypeScript**: Type safety
- **Vite**: Fast build tool with HMR
- **Tamagui**: Cross-platform UI components (future-ready for React Native)
- **TanStack Query**: Data fetching and state management
- **Zod**: Runtime schema validation

## Future Extensions

This web client is designed to be easily portable to other platforms:

- **Desktop**: Electron or Tauri wrapper
- **Mobile**: React Native or Expo (leveraging Tamagui's cross-platform support)

## Prerequisites

- Node.js 18+ and npm
- Running TwelveReader server (default: `http://localhost:8080`)

## Installation

```bash
npm install
```

## Development

```bash
npm run dev
```

The web client will start on `http://localhost:3000` and proxy API requests to `http://localhost:8080`.

## Building for Production

```bash
npm run build
```

The production build will be in the `dist` directory.

## Preview Production Build

```bash
npm run preview
```

## Usage

1. **Start the TwelveReader server** (see server README)
2. **Start the web client**: `npm run dev`
3. **Upload a book**: Click "Upload Book" and select a file
4. **Monitor processing**: Switch to "View Status" to track progress
5. **Play the book**: Once synthesized, use "Play Book" to listen with synchronized text

## API Integration

The web client communicates with the TwelveReader server API:

- `POST /api/v1/books` - Upload books
- `GET /api/v1/books/:id` - Get book metadata
- `GET /api/v1/books/:id/status` - Get processing status
- `GET /api/v1/books/:id/segments` - Get book segments
- `GET /api/v1/books/:id/audio/:segmentId` - Stream audio
- `GET /api/v1/books/:id/download` - Download full package

See `API.md` in the repository root for complete API documentation.

## Project Structure

```
src/
├── api/              # API client and React Query hooks
│   ├── client.ts     # API request functions
│   └── hooks.ts      # React Query hooks
├── components/       # React components
│   ├── BookUpload.tsx
│   ├── BookStatusCard.tsx
│   └── BookPlayer.tsx
├── types/            # TypeScript types and Zod schemas
│   └── api.ts
├── tamagui.config.ts # Tamagui configuration
├── App.tsx           # Main application component
└── main.tsx          # Application entry point
```

## Development Notes

- The dev server proxies `/api` and `/health` requests to the backend server
- Auto-refresh is enabled for processing status when books are being processed
- Audio playback automatically advances to the next segment
- All API responses are validated with Zod schemas for type safety

## Troubleshooting

### Server Connection Issues

If the web client cannot connect to the server:

1. Ensure the TwelveReader server is running on port 8080
2. Check the proxy configuration in `vite.config.ts`
3. Verify CORS is properly configured on the server

### Build Issues

If you encounter build errors:

1. Clear node_modules and reinstall: `rm -rf node_modules package-lock.json && npm install`
2. Ensure you're using Node.js 18 or later
3. Check for TypeScript errors: `npm run build`

## License

See LICENSE file in repository root.
