import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { render } from "vitest-browser-react";

const ghosttyMocks = vi.hoisted(() => {
  const init = vi.fn(async () => {});

  class FitAddon {
    public static instances: FitAddon[] = [];
    public fit = vi.fn();
    public observeResize = vi.fn();
    public dispose = vi.fn();

    public constructor() {
      FitAddon.instances.push(this);
    }
  }

  class Terminal {
    public static instances: Terminal[] = [];
    public static createdOptions: unknown[] = [];
    public openedElement: Element | null = null;
    public writeCalls: string[] = [];
    public onDataHandler: ((data: string) => void) | null = null;
    public onResizeHandler: ((size: { cols: number; rows: number }) => void) | null = null;
    public loadAddon = vi.fn();
    public focus = vi.fn();
    public dispose = vi.fn();
    public clear = vi.fn();
    public cols = 80;
    public rows = 24;

    public constructor(options?: unknown) {
      Terminal.createdOptions.push(options);
      Terminal.instances.push(this);
    }

    public open(element: Element): void {
      this.openedElement = element;
    }

    public onData(handler: (data: string) => void): { dispose: () => void } {
      this.onDataHandler = handler;
      return {
        dispose: () => {
          this.onDataHandler = null;
        },
      };
    }

    public onResize(handler: (size: { cols: number; rows: number }) => void): { dispose: () => void } {
      this.onResizeHandler = handler;
      return {
        dispose: () => {
          this.onResizeHandler = null;
        },
      };
    }

    public write(data: string): void {
      this.writeCalls.push(data);
    }

    public emitData(data: string): void {
      this.onDataHandler?.(data);
    }

    public emitResize(cols: number, rows: number): void {
      this.cols = cols;
      this.rows = rows;
      this.onResizeHandler?.({ cols, rows });
    }
  }

  return { init, FitAddon, Terminal };
});

vi.mock("ghostty-web", () => ({
  init: ghosttyMocks.init,
  Terminal: ghosttyMocks.Terminal,
  FitAddon: ghosttyMocks.FitAddon,
}));

import App from "./App";

class MockWebSocket {
  public static readonly CONNECTING = 0;
  public static readonly OPEN = 1;
  public static readonly CLOSING = 2;
  public static readonly CLOSED = 3;

  public static instances: MockWebSocket[] = [];

  public readonly url: string;
  public readyState = MockWebSocket.CONNECTING;
  public onopen: ((event: Event) => void) | null = null;
  public onmessage: ((event: MessageEvent<string>) => void) | null = null;
  public onclose: ((event: CloseEvent) => void) | null = null;
  public onerror: ((event: Event) => void) | null = null;
  public sent: string[] = [];

  public constructor(url: string | URL) {
    this.url = String(url);
    MockWebSocket.instances.push(this);
  }

  public close(): void {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.(new CloseEvent("close"));
  }

  public send(data: string): void {
    this.sent.push(data);
  }

  public emitOpen(): void {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.(new Event("open"));
  }

  public emitMessage(data: string): void {
    this.onmessage?.({ data } as MessageEvent<string>);
  }

  public emitClose(): void {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.(new CloseEvent("close"));
  }
}

describe("App", () => {
  beforeEach(() => {
    ghosttyMocks.init.mockClear();
    ghosttyMocks.Terminal.instances = [];
    ghosttyMocks.Terminal.createdOptions = [];
    ghosttyMocks.FitAddon.instances = [];
    MockWebSocket.instances = [];
    vi.stubGlobal("WebSocket", MockWebSocket as unknown as typeof WebSocket);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  test("terminalのみを前面表示し ghostty を初期化する", async () => {
    const screen = await render(<App />);
    await expect.element(screen.getByTestId("terminal-root")).toBeInTheDocument();
    await vi.waitFor(() => {
      expect(ghosttyMocks.init).toHaveBeenCalledTimes(1);
      expect(ghosttyMocks.Terminal.instances).toHaveLength(1);
      expect(ghosttyMocks.FitAddon.instances).toHaveLength(1);
    });
    expect(ghosttyMocks.FitAddon.instances[0]?.fit).toHaveBeenCalledTimes(1);
    expect(ghosttyMocks.FitAddon.instances[0]?.observeResize).toHaveBeenCalledTimes(1);
    expect(ghosttyMocks.Terminal.instances[0]?.openedElement).not.toBeNull();
    expect(ghosttyMocks.Terminal.createdOptions[0]).toMatchObject({
      fontFamily: expect.stringContaining("Nerd Font"),
    });
  });

  test("受信データを描画し、キー入力とresizeをWebSocketへ送る", async () => {
    await render(<App />);
    await vi.waitFor(() => {
      expect(ghosttyMocks.Terminal.instances).toHaveLength(1);
      expect(MockWebSocket.instances).toHaveLength(1);
    });

    const terminal = ghosttyMocks.Terminal.instances[0];
    const socket = MockWebSocket.instances[0];
    if (!terminal || !socket) {
      throw new Error("terminal or websocket not initialized");
    }

    socket.emitOpen();
    socket.emitMessage("hello");
    expect(terminal.writeCalls).toContain("hello");

    terminal.emitData("pwd\n");
    expect(socket.sent).toContain("pwd\n");

    terminal.emitResize(132, 43);
    expect(socket.sent).toContain('{"type":"resize","cols":132,"rows":43}');
  });

  test("切断時にdialogを表示し、再接続後にconsoleをクリアする", async () => {
    const screen = await render(<App />);
    await vi.waitFor(() => {
      expect(ghosttyMocks.Terminal.instances).toHaveLength(1);
      expect(MockWebSocket.instances).toHaveLength(1);
    });

    const terminal = ghosttyMocks.Terminal.instances[0];
    const firstSocket = MockWebSocket.instances[0];
    if (!terminal || !firstSocket) {
      throw new Error("terminal or socket not initialized");
    }

    firstSocket.emitOpen();
    firstSocket.emitClose();

    await expect.element(screen.getByText("WebSocket接続が切断されました。")).toBeInTheDocument();
    await expect.element(screen.getByText("再接続を試行しています…")).toBeInTheDocument();

    await vi.waitFor(
      () => {
        expect(MockWebSocket.instances.length).toBeGreaterThanOrEqual(2);
      },
      { timeout: 3_000 },
    );

    const secondSocket = MockWebSocket.instances[1];
    if (!secondSocket) {
      throw new Error("second socket not initialized");
    }
    secondSocket.emitOpen();

    await vi.waitFor(() => {
      expect(terminal.clear).toHaveBeenCalledTimes(1);
      expect(document.body.textContent ?? "").not.toContain("WebSocket接続が切断されました。");
    });
  });
});
