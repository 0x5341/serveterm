import { describe, expect, test, vi } from "vitest";
import { createTerminalClient, toWebSocketURL, type SocketLike } from "./terminal-client";

class MockSocket implements SocketLike {
  public static readonly CONNECTING = 0;
  public static readonly OPEN = 1;
  public static readonly CLOSING = 2;
  public static readonly CLOSED = 3;

  public readyState = MockSocket.CONNECTING;
  public onopen: ((event: Event) => void) | null = null;
  public onmessage: ((event: MessageEvent<string>) => void) | null = null;
  public onclose: ((event: CloseEvent) => void) | null = null;
  public onerror: ((event: Event) => void) | null = null;
  public sent: string[] = [];

  public close(): void {
    this.readyState = MockSocket.CLOSED;
    this.onclose?.(new CloseEvent("close"));
  }

  public send(data: string): void {
    this.sent.push(data);
  }

  public emitOpen(): void {
    this.readyState = MockSocket.OPEN;
    this.onopen?.(new Event("open"));
  }

  public emitMessage(data: string): void {
    this.onmessage?.({ data } as MessageEvent<string>);
  }
}

describe("toWebSocketURL", () => {
  test("http は ws に変換される", () => {
    expect(toWebSocketURL({ protocol: "http:", host: "localhost:8080" })).toBe(
      "ws://localhost:8080/ws",
    );
  });

  test("https は wss に変換される", () => {
    expect(toWebSocketURL({ protocol: "https:", host: "example.com" })).toBe(
      "wss://example.com/ws",
    );
  });
});

describe("createTerminalClient", () => {
  test("接続状態と入出力イベントを扱える", () => {
    const statuses: string[] = [];
    const outputs: string[] = [];
    const socket = new MockSocket();

    const client = createTerminalClient({
      createSocket: () => socket,
      location: { protocol: "http:", host: "localhost:8080" },
      onStatusChange: (status) => statuses.push(status),
      onOutput: (chunk) => outputs.push(chunk),
    });

    socket.emitOpen();
    socket.emitMessage("hello");
    const sent = client.send("pwd\n");
    client.close();

    expect(statuses).toEqual(["connecting", "connected", "disconnected"]);
    expect(outputs).toEqual(["hello"]);
    expect(sent).toBe(true);
    expect(socket.sent).toEqual(["pwd\n"]);
  });

  test("未接続状態では send は false を返す", () => {
    const socket = new MockSocket();
    const onOutput = vi.fn();
    const onStatusChange = vi.fn();

    const client = createTerminalClient({
      createSocket: () => socket,
      location: { protocol: "http:", host: "localhost:8080" },
      onStatusChange,
      onOutput,
    });

    expect(client.send("x")).toBe(false);
  });

  test("切断後は再接続を試行する", async () => {
    const statuses: string[] = [];
    const sockets: MockSocket[] = [];

    const client = createTerminalClient({
      createSocket: () => {
        const socket = new MockSocket();
        sockets.push(socket);
        return socket;
      },
      location: { protocol: "http:", host: "localhost:8080" },
      onStatusChange: (status) => statuses.push(status),
      onOutput: vi.fn(),
      reconnectDelayMs: 1,
    });

    const first = sockets[0];
    if (!first) {
      throw new Error("first socket not created");
    }
    first.emitOpen();
    first.close();

    await vi.waitFor(() => {
      expect(sockets).toHaveLength(2);
    });

    const second = sockets[1];
    if (!second) {
      throw new Error("second socket not created");
    }
    second.emitOpen();

    expect(statuses).toEqual(["connecting", "connected", "disconnected", "connecting", "connected"]);

    client.close();
  });
});
