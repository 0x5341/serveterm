# serveterm: terminalをwebuiで使う

# 目標
- terminalをwebuiから実行する

# 非目標
- terminal multiplexer
  - tmux, zellijなど

# 技術スタック
- backend
  - go
  - embed.FS(ui埋め込み)
  - gorilla/websocket
- frontend
  - vite
  - react
  - vitest
  - vitest browser mode
  - tailwindcss
  - shadcn/ui
  - coder/ghostty-web

# フォルダ構成
```
/ <- go backend
/ui <- react frontend
```

# 注意事項
- 必ずテストを記述すること
  - vitest, vitest browser mode, go test
