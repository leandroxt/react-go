import { useAppSelector, useAppDispatch } from "../../app/hooks";
import { selectLoading } from "./stocks.select";
import { Loading } from "../shared/Loading";
import { useWebSocketContext } from "../../context/websocket.context";
import { getStockPrices } from "./stocks.slice";
// import { useEffect } from "react";

export function Stocks() {
  const dispatch = useAppDispatch();
  const { webSocketMessage } = useWebSocketContext();

  const loading = useAppSelector(selectLoading);

  function onSendMessage() {
    dispatch(getStockPrices());
  }

  // useEffect(() => {
  //   console.log("chamou", webSocketMessage?.data);
  //   setStocks(JSON.parse(webSocketMessage?.data || "[]"));
  // }, [webSocketMessage?.data]);

  return (
    <div className="flex justify-center items-center h-96">
      <div className="flex justify-center">
        <div>
          <button
            className="bg-gray-300 hover:bg-gray-400 text-gray-800 font-bold py-2 px-4 rounded inline-flex items-center"
            onClick={onSendMessage}
          >
            {/* <svg
              className="fill-current w-4 h-4 mr-2"
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 20 20"
            >
              <path d="M13 8V2H7v6H2l8 8 8-8h-5zM0 18h20v2H0v-2z" />
            </svg> */}
            <span>Generate</span>
          </button>
          {loading && <Loading />}
          {webSocketMessage?.data}
        </div>
      </div>
    </div>
  );
}
