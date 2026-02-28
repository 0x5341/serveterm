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
};

export type TerminalClient = {
  send(input: string): boolean;
  close(): void;
};

const SOCKET_OPEN = 1;

export function toWebSocketURL(location: LocationLike): string {
  const protocol = location.protocol === "https:" ? "wss" : "ws";
  return `${protocol}://${location.host}/ws`;
}

export function createTerminalClient(options: TerminalClientOptions): TerminalClient {
  const location = options.location ?? window.location;
  const createSocket = options.createSocket ?? ((url: string) => new WebSocket(url));
  const socket = createSocket(toWebSocketURL(location));

  options.onStatusChange("connecting");

  socket.onopen = () => {
    options.onStatusChange("connected");
  };
  socket.onmessage = (event: MessageEvent<string>) => {
    options.onOutput(String(event.data ?? ""));
  };
  socket.onerror = () => {
    options.onStatusChange("error");
  };
  socket.onclose = () => {
    options.onStatusChange("disconnected");
  };

  return {
    send(input: string): boolean {
      if (socket.readyState !== SOCKET_OPEN) {
        return false;
      }
      socket.send(input);
      return true;
    },
    close(): void {
      socket.close();
    },
  };
}
