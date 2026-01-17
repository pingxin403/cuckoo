# Web Frontend Application

React + TypeScript frontend application for the Monorepo Hello/TODO Services.

## Technology Stack

- **React 19** - UI framework
- **TypeScript** - Type safety
- **Vite** - Build tool and dev server
- **TanStack Query (React Query)** - Data fetching and caching
- **gRPC-Web** - Communication with backend services
- **ts-proto** - TypeScript code generation from Protobuf

## Project Structure

```
apps/web/
├── src/
│   ├── components/        # React components
│   │   ├── HelloForm.tsx  # Hello service UI
│   │   ├── TodoForm.tsx   # TODO creation form
│   │   └── TodoList.tsx   # TODO list display
│   ├── hooks/             # Custom React hooks
│   │   └── useTodos.ts    # TODO CRUD operations
│   ├── services/          # gRPC client wrappers
│   │   ├── helloClient.ts # Hello service client
│   │   └── todoClient.ts  # TODO service client
│   ├── gen/               # Generated TypeScript code from Protobuf
│   │   ├── hello.ts
│   │   └── todo.ts
│   ├── App.tsx            # Main application component
│   └── main.tsx           # Application entry point
├── package.json
├── tsconfig.json
├── vite.config.ts
└── README.md
```

## Development

### Prerequisites

- Node.js 18+ and npm
- Backend services running (Hello service on port 9090, TODO service on port 9091)
- Envoy proxy running on port 8080 (for gRPC-Web translation)

### Install Dependencies

```bash
npm install
```

### Generate Protobuf Code

Generate TypeScript code from Protobuf definitions:

```bash
npm run gen-proto
```

This reads from `../../api/v1/*.proto` and generates code in `src/gen/`.

### Start Development Server

```bash
npm run dev
```

The application will be available at http://localhost:5173

### Proxy Configuration

The Vite dev server is configured to proxy API requests to the Envoy gateway:

- `/api/hello/*` → `http://localhost:8080` (Hello Service)
- `/api/todo/*` → `http://localhost:8080` (TODO Service)

This matches the production routing pattern through Higress.

## Building for Production

```bash
npm run build
```

The production build will be output to the `dist/` directory.

## Features

### Hello Service Integration

- Input field for name
- Calls Hello service via gRPC-Web
- Displays greeting response
- Error handling with user-friendly messages

### TODO Management

- Create new TODO items with title and description
- List all TODO items
- Update TODO items (edit title, description, toggle completion)
- Delete TODO items
- Real-time updates using React Query
- Loading states and error handling

## Architecture

### Communication Flow

```
Browser (React App)
    ↓ HTTP/gRPC-Web
Vite Dev Proxy (localhost:5173)
    ↓ Proxy to localhost:8080
Envoy Gateway (localhost:8080)
    ↓ gRPC
Backend Services (Hello: 9090, TODO: 9091)
```

### State Management

- **TanStack Query** for server state (API data)
- **React useState** for local UI state
- Automatic cache invalidation after mutations
- Optimistic updates for better UX

### Type Safety

- Full TypeScript coverage
- Generated types from Protobuf definitions
- Type-safe API calls
- Compile-time error checking

## Testing

```bash
npm test
```

## Linting

```bash
npm run lint
```

## Deployment

For detailed information about how the frontend communicates with backend services in different environments (local, testing, production), see:

**[DEPLOYMENT.md](./DEPLOYMENT.md)** - 前后端通信架构详细说明

Key points:
- **Local Development**: Vite proxy → Envoy (localhost:8080) → Backend services
- **Testing/Production**: CDN/Nginx → Higress Ingress → Backend services (K8s)
- All environments use the same API paths (`/api/hello`, `/api/todo`)

## Notes

- The frontend expects the Envoy proxy (local) or Higress gateway (production) to handle gRPC-Web to gRPC translation
- All API calls go through the proxy at `/api/hello` and `/api/todo`
- The proxy configuration in `vite.config.ts` is for development only
- In production, Higress Ingress handles the routing
