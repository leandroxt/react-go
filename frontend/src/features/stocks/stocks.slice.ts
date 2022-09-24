import { createAsyncThunk, createSlice, PayloadAction } from "@reduxjs/toolkit";

export interface Stock {
  symbol: string;
  shortName: string;
  regularMarketPrice: number;
}

interface stocksState {
  loading: boolean;
  tickers: Stock[];
}

const initialState: stocksState = {
  loading: false,
  tickers: [],
};

// http://localhost:8080/v1/greetings/leandro
export const getStockPrices = createAsyncThunk(
  "stocks/getStockPrices",
  async (): Promise<string> => {
    const response = await fetch(`http://localhost:8080/v1/stocks/price`);
    return await response.json();
  }
);

export const stocksSlice = createSlice({
  name: "stocks",
  initialState,
  reducers: {
    setStocks(state, action: PayloadAction<unknown>) {
      if (!action.payload) {
        return;
      }
      const { results } = action.payload as { results: Stock[] };
      state.tickers = results;
    },
  },
  extraReducers(builder) {
    builder
      .addCase(getStockPrices.pending, (state) => {
        state.loading = true;
      })
      .addCase(getStockPrices.fulfilled, (state, action) => {
        state.loading = false;
      })
      .addCase(getStockPrices.rejected, (state, action) => {
        state.loading = false;
      });
  },
});

export const { setStocks } = stocksSlice.actions;

export default stocksSlice.reducer;
