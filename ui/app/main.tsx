import { createElement, type ComponentType } from 'react';
import { createRoot } from 'react-dom/client';

import { ConnectWallet } from './islands/ConnectWallet';
import { DepositForm } from './islands/DepositForm';

// The registry. Adding an island means adding a line here and using
// {{island "name" .Props}} in a template — nothing else mounts React.
//
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const islands: Record<string, ComponentType<any>> = {
  'connect-wallet': ConnectWallet,
  'deposit-form': DepositForm,
};

for (const el of document.querySelectorAll<HTMLElement>('[data-island]')) {
  const name = el.dataset.island!;
  const Component = islands[name];

  if (!Component) {
    console.error(`island "${name}" is not in the registry`);
    continue;
  }

  // Props were written by the Go `island` template func. Read them before
  // render(), which replaces the mount point's children.
  const json = el.querySelector('script[type="application/json"]')?.textContent;
  const props = json ? JSON.parse(json) : {};

  createRoot(el).render(createElement(Component, props));
}
