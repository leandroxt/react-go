import { createSelector } from "@reduxjs/toolkit";
import { RootState } from "../../app/store";

const selectStocks = (state: RootState) => state.stocks;

export const selectLoading = createSelector(
  selectStocks,
  ({ loading }) => loading || false
);

export const selectTickers = createSelector(
  selectStocks,
  ({ tickers }) => tickers || []
);
