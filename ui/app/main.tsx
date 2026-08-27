import { useState } from 'react';
import { createRoot } from 'react-dom/client';

/** Mirrors the Go `state` struct in cmd/web/main.go. */
type State = { count: number };

function Counter({ start }: { start: number }) {
  const [count, setCount] = useState(start);

  return (
    <div className="counter">
      <button onClick={() => setCount(count - 1)}>−</button>
      <output>{count}</output>
      <button onClick={() => setCount(count + 1)}>+</button>
    </div>
  );
}

// Server state arrives as JSON in the page, not via a fetch.
const json = document.getElementById('state')!.textContent!;
const { count }: State = JSON.parse(json);

createRoot(document.getElementById('root')!).render(<Counter start={count} />);
