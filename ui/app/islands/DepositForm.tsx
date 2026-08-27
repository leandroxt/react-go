import { useEffect, useRef, useState } from 'react';
import { connect, errorMessage, shortAddress, useWallet } from '../wallet';

/** Props come from the Go handler via {{island "deposit-form" .Data.Deposit}}. */
type Props = {
  poolId: string;
  pair: string;
  tokenA: string;
  tokenB: string;
  minDeposit: number;
  chainId: number;
};

export function DepositForm({
  poolId,
  pair,
  tokenA,
  tokenB,
  minDeposit,
  chainId,
}: Props) {
  const { account } = useWallet();
  const dialog = useRef<HTMLDialogElement>(null);

  const [amount, setAmount] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [receipt, setReceipt] = useState<string | null>(null);

  // Native <dialog> is imperative, so opening is an effect rather than JSX.
  const [open, setOpen] = useState(false);
  useEffect(() => {
    const el = dialog.current;
    if (!el) return;
    if (open && !el.open) el.showModal();
    if (!open && el.open) el.close();
  }, [open]);

  const value = Number(amount);
  const invalid =
    amount === '' || Number.isNaN(value) || value < minDeposit
      ? `Valor mínimo: ${minDeposit} ${tokenA}.`
      : null;

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (invalid) return;

    setBusy(true);
    setError(null);
    try {
      const hash = await createPosition({ poolId, amount: value, chainId });
      setReceipt(hash);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  }

  if (!account) {
    return (
      <div className="deposit">
        <p className="muted">Conecte a carteira para montar uma posição nesta pool.</p>
        <button className="btn btn--primary" onClick={() => void connect(chainId)}>
          Conectar carteira
        </button>
      </div>
    );
  }

  return (
    <div className="deposit">
      <p className="muted">
        Conectado como <strong>{shortAddress(account)}</strong>
      </p>
      <button className="btn btn--primary" onClick={() => setOpen(true)}>
        Montar posição em {pair}
      </button>

      <dialog ref={dialog} className="modal" onClose={() => setOpen(false)}>
        <form onSubmit={onSubmit}>
          <h3>Adicionar liquidez · {pair}</h3>

          {receipt ? (
            <>
              <p className="ok">Posição simulada com sucesso.</p>
              <p className="muted mono">{receipt}</p>
              <p className="muted">
                Simulação: nenhuma transação foi enviada à rede.
              </p>
              <menu>
                <button type="button" className="btn" onClick={() => setOpen(false)}>
                  Fechar
                </button>
              </menu>
            </>
          ) : (
            <>
              <label htmlFor="amount">Valor em {tokenA}</label>
              <input
                id="amount"
                inputMode="decimal"
                autoComplete="off"
                placeholder={String(minDeposit)}
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
              />
              <p className="muted">
                O par será formado com {tokenB} na proporção atual da pool.
              </p>

              {amount !== '' && invalid && <p className="err">{invalid}</p>}
              {error && <p className="err">{error}</p>}

              <menu>
                <button
                  type="button"
                  className="btn"
                  onClick={() => setOpen(false)}
                  disabled={busy}
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  className="btn btn--primary"
                  disabled={busy || invalid !== null}
                >
                  {busy ? 'Assinando…' : 'Confirmar'}
                </button>
              </menu>
            </>
          )}
        </form>
      </dialog>
    </div>
  );
}

/**
 * SEAM — not implemented.
 *
 * The real flow is: POST to the Go backend, get a 402 with the x402 payment
 * requirements, pay in USDC on Base, retry with the payment header, then have
 * the wallet sign the position transaction the server built.
 *
 * Left as a stub on purpose: implementing it means committing to the x402
 * header format and to real contract addresses, and guessing at either would
 * be worse than an honest gap.
 */
async function createPosition(args: {
  poolId: string;
  amount: number;
  chainId: number;
}): Promise<string> {
  await new Promise((r) => setTimeout(r, 600));
  return `simulado:${args.poolId}:${args.amount}@${args.chainId}`;
}
