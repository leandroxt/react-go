import { useState } from 'react';
import { connect, errorMessage, shortAddress, useWallet } from '../wallet';

/** Props come from the Go handler via {{island "connect-wallet" .Wallet}}. */
type Props = {
  chainId: number;
  chainName: string;
};

export function ConnectWallet({ chainId, chainName }: Props) {
  const { account } = useWallet();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onConnect() {
    setBusy(true);
    setError(null);
    try {
      await connect(chainId);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  }

  if (account) {
    return (
      <span className="wallet wallet--on" title={account}>
        <span className="dot" /> {shortAddress(account)}
      </span>
    );
  }

  return (
    <span className="wallet">
      <button className="btn" onClick={onConnect} disabled={busy}>
        {busy ? 'Conectando…' : `Conectar na ${chainName}`}
      </button>
      {error && <span className="wallet-error">{error}</span>}
    </span>
  );
}
