# Keel UI

Keel's dashboard is a client-rendered React application built with Vite and shadcn/ui. Its production output is entirely static and is copied into `/www` in the Keel container; no Node.js process is used in production.

## Development

Run Keel on port 9300, then start the Vite development server. Vite proxies the existing `/v1` API to Keel.

```sh
cd ui
npm ci
npm run dev
```

Open `http://localhost:8000`. To exercise the production bundle through Keel instead:

```sh
cd ui
npm run build
cd ..
go run ./cmd/keel --no-incluster --ui-dir ui/dist
```

Set Keel's existing `BASIC_AUTH_USER` and `BASIC_AUTH_PASSWORD` environment variables to enable the dashboard and admin API.

## Checks

```sh
npm run typecheck
npm run lint
npm test
npm run build
```

`npm run build` writes only static files to `ui/dist`. Browser history routes such as `/approvals` are served by Keel's `index.html` fallback.

## shadcn initialization

Initialized on 2026-08-04 with stable CLI `shadcn@4.16.1`:

```sh
npx shadcn@4.16.1 init --defaults --template vite --name ui-react --no-monorepo
```

The generated defaults are recorded in `components.json`: `base-nova`, Base UI, neutral CSS variables, Geist, Lucide, Tailwind CSS 4, React 19, TypeScript, and Vite. The sole scaffold deviation is `--template vite` instead of the CLI's default Next.js template, because Keel requires a static SPA with no SSR or production Node.js runtime.

The resolved scaffold baseline is `shadcn@4.16.1`, `react@19.2.8`, `react-dom@19.2.8`, `@base-ui/react@1.6.0`, `tailwindcss@4.3.3`, `@tailwindcss/vite@4.3.3`, `vite@8.2.0`, `typescript@6.0.3`, `lucide-react@1.28.0`, `class-variance-authority@0.7.1`, `clsx@2.1.1`, `tailwind-merge@3.6.0`, and `tw-animate-css@1.4.0`. `package-lock.json` records these and every transitive dependency exactly.
