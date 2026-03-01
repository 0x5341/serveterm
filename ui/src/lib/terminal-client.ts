export type TerminalStatus = "connecting" | "connected" | "disconnected" | "error";

export type LocationLike = {
  protocol: string;
  host: string;
};

export type SocketLike = {
  readyState: number;
  onopen: ((event: Event) => void) | null;
  onmessage: ((event: MessageEvent<string>) => void) | null;
  onclose: ((event: CloseEvent) => void) | null;
  onerror: ((event: Event) => void) | null;
  send(data: string): void;
  close(): void;
};

type TerminalClientOptions = {
  location?: LocationLike;
  createSocket?: (url: string) => SocketLike;
  onOutput: (chunk: string) => void;
  onStatusChange: (status: TerminalStatus) => void;
  reconnectDelayMs?: number;
};

export type TerminalClient = {
  send(input: string): boolean;
  close(): void;
};

const SOCKET_OPEN = 1;
const DEFAULT_RECONNECT_DELAY_MS = 1_000;

export function toWebSocketURL(location: LocationLike): string {
  const protocol = location.protocol === "https:" ? "wss" : "ws";
  return `${protocol}://${location.host}/ws`;
}

export function createTerminalClient(options: TerminalClientOptions): TerminalClient {
  const location = options.location ?? window.location;
  const createSocket = options.createSocket ?? ((url: string) => new WebSocket(url));
  const reconnectDelayMs = options.reconnectDelayMs ?? DEFAULT_RECONNECT_DELAY_MS;
  const wsURL = toWebSocketURL(location);
  let socket: SocketLike | null = null;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  let closed = false;

  const connect = () => {
    if (closed) {
      return;
    }
    options.onStatusChange("connecting");
    const currentSocket = createSocket(wsURL);
    socket = currentSocket;

    currentSocket.onopen = () => {
      if (closed || socket !== currentSocket) {
        return;
      }
      options.onStatusChange("connected");
    };
    currentSocket.onmessage = (event: MessageEvent<string>) => {
      if (closed || socket !== currentSocket) {
        return;
      }
      options.onOutput(String(event.data ?? ""));
    };
    currentSocket.onerror = () => {
      if (closed || socket !== currentSocket) {
        return;
      }
      options.onStatusChange("error");
    };
    currentSocket.onclose = () => {
      if (socket !== currentSocket) {
        return;
      }
      options.onStatusChange("disconnected");
      if (closed) {
        return;
      }
      reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        connect();
      }, reconnectDelayMs);
    };
  };
  connect();

  return {
    send(input: string): boolean {
      if (!socket) {
        return false;
      }
      if (socket.readyState !== SOCKET_OPEN) {
        return false;
      }
      socket.send(input);
      return true;
    },
    close(): void {
      closed = true;
      if (reconnectTimer !== null) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
      }
      socket?.close();
    },
  };
}
