# Outway Admin UI

Modern, fast, and minimal admin interface for Outway DNS proxy, built with Preact + TypeScript + Vite.

## Features

- **WebSocket-first architecture** with REST fallback
- **Real-time updates** for stats, rules, upstreams, and query history
- **Interactive charts** for RPS and response time monitoring
- **CRUD operations** for rules and upstreams management
- **Virtualized history table** for performance
- **Responsive design** with clean, minimal UI
- **TypeScript strict mode** for type safety

## Tech Stack

- **Preact** - Lightweight React alternative
- **TypeScript** - Type safety and better DX
- **Vite** - Fast build tool and dev server
- **Zustand** - Simple state management
- **Chart.js** - Interactive charts
- **CSS Custom Properties** - Design system tokens

## Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Lint code
npm run lint

# Fix linting issues
npm run lint:fix

# Type check
npm run type-check
```

## Project Structure

```
ui/
├── src/
│   ├── app/           # App routing and main component
│   ├── providers/     # WebSocket, REST, and failover providers
│   ├── store/         # Zustand state management
│   ├── components/    # Reusable UI components
│   ├── pages/         # Page components (Overview, Rules, etc.)
│   ├── charts/        # Chart utilities and hooks
│   ├── styles/        # CSS tokens and component styles
│   └── utils/         # Utility functions
├── dist/              # Built assets (embedded by Go)
└── package.json       # Dependencies and scripts
```

## Architecture

### Data Flow
1. **FailoverProvider** orchestrates WebSocket (primary) and REST (fallback)
2. **Zustand stores** manage state for rules, upstreams, stats, history
3. **Components** subscribe to store updates via selectors
4. **Real-time updates** via WebSocket with automatic reconnection

### WebSocket Protocol
- Messages: `{ type: 'stats'|'history'|'rules'|'upstreams', data: any }`
- Auto-reconnect with exponential backoff
- Initial snapshots on connection

### REST API
- `GET /api/rules` - Fetch rules
- `POST /api/rules` - Create/update rule
- `DELETE /api/rules` - Delete rule
- `GET /api/upstreams` - Fetch upstreams
- `POST /api/upstreams` - Update upstreams
- `GET /api/stats` - Fetch statistics
- `GET /api/history` - Fetch query history

## Performance

- **Bundle size**: ~80-120KB gzipped target
- **Virtualized tables** for large datasets
- **Memoized selectors** to prevent unnecessary re-renders
- **Optimistic updates** with server reconciliation
- **Chart instance reuse** to avoid recreation

## Design System

### CSS Tokens
```css
:root {
  --bg: #ffffff;
  --card: #ffffff;
  --fg: #111827;
  --muted: #6b7280;
  --accent: #111111;
  --ok: #16a34a;
  --bad: #dc2626;
  --bd: #e5e7eb;
  --shadow: 0 6px 24px rgba(17, 17, 17, 0.06);
}
```

### Components
- **Button** - Primary/secondary variants with sizes
- **Input** - Form inputs with validation states
- **Select** - Dropdown selects with options
- **Tabs** - Navigation tabs with active states
- **Card** - Content containers with consistent styling
- **Table** - Data tables with hover states

## Integration

The UI is built into `dist/` and embedded by the Go server using `go:embed`. The Go server serves:
- Static assets from `dist/static/`
- `index.html` for all non-API routes (SPA routing)

## Building

```bash
# From project root
make ui-build    # Build UI
make build       # Build UI + Go binary
```

## Browser Support

Modern evergreen browsers with ES2020 support:
- Chrome 80+
- Firefox 75+
- Safari 13+
- Edge 80+
