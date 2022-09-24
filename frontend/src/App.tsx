import { useWebSocketContext } from "./context/websocket.context";
import { Ping } from "./features/shared/Ping";
import { Stocks } from "./features/stocks/Stocks";

function App() {
  const { webSocketState } = useWebSocketContext();

  return (
    <div className="min-h-full">
      <header className="bg-white shadow">
        <div className="mx-auto max-w-7xl py-6 px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between w-full">
            <h1 className="text-3xl font-bold tracking-tight text-gray-900">
              Dashboard
            </h1>
            <div>
              <Ping
                color={webSocketState === "ON" ? "bg-green-600" : "bg-red-600"}
              />
            </div>
          </div>
        </div>
      </header>

      <main>
        <div className="mx-auto max-w-7xl py-6 sm:px-6 lg:px-8">
          <div className="px-4 py-6 sm:px-0">
            <div className="h-96 rounded-lg border-4 border-dashed border-gray-200">
              <Stocks />
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}

export default App;
