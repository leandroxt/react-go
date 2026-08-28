import { StrictMode, type ComponentType } from 'react';
import { createRoot } from 'react-dom/client';

import { SharedProvider, type Shared } from './shared';

/** What the Go `state` template func writes into the page. */
type State<P> = {
  shared: Shared;
  props: P;
};

/**
 * Mounts a route component over <main>.
 *
 * Every route entry point ends with a call to this. Go owns the document
 * shell — <head>, header, footer — and React owns everything inside <main>,
 * so neither side has to read as half of the other.
 */
export function mount<P extends object>(Page: ComponentType<P>): void {
  const el = document.querySelector('main');
  if (!el) throw new Error('mount: no <main> in the document');

  const raw = document.getElementById('state')?.textContent;
  if (!raw) throw new Error('mount: no #state in the document');

  const { shared, props } = JSON.parse(raw) as State<P>;

  createRoot(el).render(
    <StrictMode>
      <SharedProvider value={shared}>
        <Page {...props} />
      </SharedProvider>
    </StrictMode>,
  );
}
