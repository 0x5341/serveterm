# serveterm UI

serveterm の frontend（Vite + React）です。  
terminal 出力表示・入力送信を WebSocket(`/ws`) で backend と接続します。

## 開発コマンド

```bash
pnpm dev
pnpm lint
pnpm build
pnpm vitest run
```

## ビルド成果物

- `pnpm build` の出力先は `ui/dist` です。
- Go backend は `embed.FS` で `ui/dist` を同梱して配信します。

## テスト

- `vitest` は Browser Mode（Playwright + Chromium, headless）で実行されます。
- `App.test.tsx` は UI の接続状態表示と送信操作を検証します。
- `terminal-client.test.ts` は WebSocket クライアント層を検証します。
