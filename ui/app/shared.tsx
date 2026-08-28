import { createContext, useContext } from 'react';

/** Mirrors the Go `shared` struct in cmd/web/main.go. */
export type Shared = {
  appName: string;
  user: string;
};

const SharedContext = createContext<Shared | null>(null);

export const SharedProvider = SharedContext.Provider;

/**
 * Context shared by every route.
 *
 * Pages are separate documents, so a full navigation tears down the React
 * tree — this is not client state that survives a page load. It survives
 * because Go re-injects it on every render, which also keeps the server the
 * single source of truth.
 */
export function useShared(): Shared {
  const shared = useContext(SharedContext);
  if (!shared) throw new Error('useShared called outside SharedProvider');
  return shared;
}
