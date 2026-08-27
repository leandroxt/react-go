import { useSyncExternalStore } from 'react';

/**
 * The minimum EIP-1193 surface needed to talk to MetaMask. No library: for a
 * connect button and a chain switch, wagmi/viem would be far more bundle than
 * behaviour.
 */
export interface Eip1193Provider {
  request(args: { method: string; params?: unknown[] }): Promise<unknown>;
  on(event: string, handler: (...args: any[]) => void): void;
  removeListener(event: string, handler: (...args: any[]) => void): void;
}

declare global {
  interface Window {
    ethereum?: Eip1193Provider;
  }
}

// ---------------------------------------------------------------------------
// Shared store
//
// Islands are separate React roots, so context cannot cross them. They do share
// a module graph, so a module-level store plus useSyncExternalStore lets the
// header's connect button and a page's deposit form see the same account.
// ---------------------------------------------------------------------------

export type WalletState = {
  account: string | null;
  chainId: number | null;
};

let state: WalletState = { account: null, chainId: null };
const listeners = new Set<() => void>();

function setState(next: WalletState) {
  state = next;
  for (const l of listeners) l();
}

export function useWallet(): WalletState {
  return useSyncExternalStore(
    (onChange) => {
      listeners.add(onChange);
      return () => listeners.delete(onChange);
    },
    () => state,
  );
}

// ---------------------------------------------------------------------------
// Actions
// ---------------------------------------------------------------------------

export function shortAddress(address: string): string {
  return `${address.slice(0, 6)}…${address.slice(-4)}`;
}

/** EIP-1193 rejections carry a numeric code; 4001 is "user rejected". */
export function errorMessage(err: unknown): string {
  if (typeof err === 'object' && err !== null && 'code' in err) {
    if ((err as { code: number }).code === 4001) return 'Operação recusada na carteira.';
  }
  return err instanceof Error ? err.message : String(err);
}

let listening = false;

/** Reflects wallet-side changes back into the store. Attached once. */
function listen(provider: Eip1193Provider) {
  if (listening) return;
  listening = true;

  provider.on('accountsChanged', (accounts: string[]) => {
    setState({ ...state, account: accounts[0] ?? null });
  });
  provider.on('chainChanged', (hex: string) => {
    setState({ ...state, chainId: parseInt(hex, 16) });
  });
}

export async function connect(chainId: number): Promise<void> {
  const provider = window.ethereum;
  if (!provider) {
    throw new Error('Nenhuma carteira detectada. Instale a MetaMask.');
  }

  const accounts = (await provider.request({
    method: 'eth_requestAccounts',
  })) as string[];

  // Ask the wallet to move to the chain the server told us to use.
  const hex = `0x${chainId.toString(16)}`;
  const current = (await provider.request({ method: 'eth_chainId' })) as string;
  if (current.toLowerCase() !== hex) {
    await provider.request({
      method: 'wallet_switchEthereumChain',
      params: [{ chainId: hex }],
    });
  }

  listen(provider);
  setState({ account: accounts[0] ?? null, chainId });
}
