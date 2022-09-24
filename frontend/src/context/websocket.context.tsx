import {
  createContext,
  ReactElement,
  useContext,
  useEffect,
  useState,
} from "react";

type webSocketState = "ON" | "OFF";

interface IWebSocketContext {
  webSocketSend: (message: string) => void;
  webSocketState: webSocketState;
  webSocketMessage: MessageEvent<string> | null;
}

const WebSocketContext = createContext<IWebSocketContext>(
  {} as IWebSocketContext
);

export function useWebSocketContext(): IWebSocketContext {
  return useContext(WebSocketContext);
}

interface IProps {
  children: ReactElement | ReactElement[];
}

export function WebSocketContextProvider({ children }: IProps): ReactElement {
  const [websocket, setWebsocket] = useState<WebSocket | null>(null);
  const [webSocketState, setWebSocketState] = useState<webSocketState>("OFF");
  const [webSocketMessage, setWebSocketMessage] =
    useState<MessageEvent<string> | null>(null);

  function webSocketSend(message: string): void {
    websocket?.send(message);
  }

  useEffect(() => {
    const addr = `ws://localhost:8080/ws`;
    const ws = new WebSocket(addr);

    ws.onopen = () => {
      setWebsocket(ws);
      setWebSocketState("ON");
    };

    ws.onmessage = (e: MessageEvent<string>) => {
      setWebSocketMessage(e);
    };

    ws.onerror = (e) => {
      console.error({ e });
    };

    ws.onclose = (e) => {
      console.info({ e });
      setWebSocketState("OFF");
    };
  }, []);

  return (
    <WebSocketContext.Provider
      value={{ webSocketSend, webSocketState, webSocketMessage }}
    >
      {children}
    </WebSocketContext.Provider>
  );
}
