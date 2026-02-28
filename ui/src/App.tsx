import { useEffect, useRef } from "react";
import { FitAddon, init, Terminal } from "ghostty-web";
import "./App.css";
import { createTerminalClient, type TerminalClient } from "@/lib/terminal-client";

function App() {
  const terminalRootRef = useRef<HTMLDivElement | null>(null);

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
      });

      fitAddon = new FitAddon();
      terminal.loadAddon(fitAddon);
      terminal.open(terminalRootRef.current);
      fitAddon.fit();
      fitAddon.observeResize();

      const sendResize = (cols: number, rows: number) => {
        client?.send(JSON.stringify({ type: "resize", cols, rows }));
      };

      client = createTerminalClient({
        onStatusChange: (status) => {
          if (status === "connected" && terminal) {
            sendResize(terminal.cols, terminal.rows);
          }
        },
        onOutput: (chunk) => {
          terminal?.write(chunk);
        },
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
  }, []);

  return (
    <main className="terminal-shell">
      <div ref={terminalRootRef} className="terminal-root" data-testid="terminal-root" />
    </main>
  );
}

export default App;
