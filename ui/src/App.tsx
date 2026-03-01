import { type ChangeEvent, type FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { FitAddon, init, Terminal } from "ghostty-web";
import "./App.css";
import { createTerminalClient, type TerminalClient } from "@/lib/terminal-client";
import { isThemeInputValid, parseThemeInput, readThemeCookie, writeThemeCookie } from "@/lib/theme-cookie";

function TerminalPage() {
  const terminalRootRef = useRef<HTMLDivElement | null>(null);
  const connectedOnceRef = useRef(false);
  const clearOnReconnectRef = useRef(false);
  const [showDisconnectDialog, setShowDisconnectDialog] = useState(false);
  const theme = useMemo(() => {
    const parsedTheme = parseThemeInput(readThemeCookie());
    console.log("[serveterm] parsed theme cookie", parsedTheme);
    return parsedTheme;
  }, []);

  useEffect(() => {
    let disposed = false;
    let terminal: Terminal | null = null;
    let fitAddon: FitAddon | null = null;
    let inputSubscription: { dispose: () => void } | null = null;
    let resizeSubscription: { dispose: () => void } | null = null;
    let client: TerminalClient | null = null;

    const start = async () => {
      await init();
      if (disposed || !terminalRootRef.current) {
        return;
      }

      terminal = new Terminal({
        cursorBlink: true,
        scrollback: 10_000,
        fontFamily: '"JetBrains Mono", "MesloLGS NF", "Symbols Nerd Font Mono", monospace',
        ...(theme ? { theme } : {}),
      });

      fitAddon = new FitAddon();
      terminal.loadAddon(fitAddon);
      terminal.open(terminalRootRef.current);
      fitAddon.fit();
      fitAddon.observeResize();

      const sendResize = (cols: number, rows: number) => {
        const sent = client?.send(JSON.stringify({ type: "resize", cols, rows })) ?? false;
        console.log("[serveterm] send resize", { cols, rows, sent });
      };

      client = createTerminalClient({
        onStatusChange: (status) => {
          if (disposed) {
            return;
          }
          if (status === "connected" && terminal) {
            if (clearOnReconnectRef.current) {
              terminal.clear();
              clearOnReconnectRef.current = false;
            }
            connectedOnceRef.current = true;
            setShowDisconnectDialog(false);
            sendResize(terminal.cols, terminal.rows);
            return;
          }
          if ((status === "disconnected" || status === "error") && connectedOnceRef.current) {
            clearOnReconnectRef.current = true;
            setShowDisconnectDialog(true);
          }
        },
        onOutput: (chunk) => {
          terminal?.write(chunk);
        },
        reconnectDelayMs: 1_000,
      });
      inputSubscription = terminal.onData((data) => {
        client?.send(data);
      });
      resizeSubscription = terminal.onResize(({ cols, rows }) => {
        sendResize(cols, rows);
      });
      terminal.focus();
    };

    void start();
    return () => {
      disposed = true;
      inputSubscription?.dispose();
      resizeSubscription?.dispose();
      client?.close();
      fitAddon?.dispose();
      terminal?.dispose();
    };
  }, [theme]);

  return (
    <main className="terminal-shell">
      <div className="top-hover-zone" data-testid="top-hover-zone" />
      <nav className="navigation-bar" data-testid="navigation-bar">
        <a href="/setting" target="_blank" rel="noreferrer noopener" className="navigation-link">
          Setting
        </a>
      </nav>
      <div ref={terminalRootRef} className="terminal-root" data-testid="terminal-root" />
      {showDisconnectDialog ? (
        <dialog open className="disconnect-dialog">
          <p>WebSocket接続が切断されました。</p>
          <p>再接続を試行しています…</p>
        </dialog>
      ) : null}
    </main>
  );
}

function SettingsPage() {
  const [themeInput, setThemeInput] = useState(() => readThemeCookie());
  const [errorMessage, setErrorMessage] = useState("");
  const [savedMessage, setSavedMessage] = useState("");

  const loadThemeFile = (event: ChangeEvent<HTMLInputElement>) => {
    const inputElement = event.currentTarget;
    const file = inputElement.files?.[0];
    inputElement.value = "";
    if (!file) {
      return;
    }
    void file.text().then(
      (content) => {
        setThemeInput(content);
        setErrorMessage("");
        setSavedMessage("");
      },
      () => {
        setErrorMessage("themeファイルの読み込みに失敗しました。");
        setSavedMessage("");
      },
    );
  };

  const saveTheme = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!isThemeInputValid(themeInput)) {
      setErrorMessage("themeファイル形式で入力してください。");
      setSavedMessage("");
      return;
    }
    writeThemeCookie(themeInput);
    setErrorMessage("");
    setSavedMessage("保存しました。");
  };

  return (
    <main className="settings-shell">
      <section className="settings-card">
        <h1>Theme設定</h1>
        <p>Ghosttyのthemeファイルを指定して保存してください。</p>
        <form className="settings-form" onSubmit={saveTheme}>
          <label htmlFor="theme-file-input">Themeファイル</label>
          <input
            id="theme-file-input"
            aria-label="Themeファイル"
            type="file"
            accept=".theme,.conf,.txt,text/plain"
            onChange={loadThemeFile}
          />
          <label htmlFor="theme-input">Theme file</label>
          <textarea
            id="theme-input"
            aria-label="Theme file"
            value={themeInput}
            onChange={(event) => setThemeInput(event.target.value)}
            placeholder={"palette = 0=#1b1f2a\nbackground = #1a1b26\nforeground = #a9b1d6"}
            rows={10}
          />
          <button type="submit">保存</button>
        </form>
        <a
          href="https://ghostty.org/docs/features/theme"
          target="_blank"
          rel="noreferrer noopener"
          className="settings-link"
        >
          Ghosttyのtheme一覧
        </a>
        {errorMessage ? (
          <p role="alert" className="settings-error">
            {errorMessage}
          </p>
        ) : null}
        {savedMessage ? <p className="settings-success">{savedMessage}</p> : null}
      </section>
    </main>
  );
}

function App() {
  const pathname = window.location.pathname.replace(/\/+$/, "") || "/";
  if (pathname === "/setting") {
    return <SettingsPage />;
  }
  return <TerminalPage />;
}

export default App;
