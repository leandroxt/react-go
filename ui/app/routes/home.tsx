import { useState } from 'react';

import { mount } from '../mount';
import { useShared } from '../shared';

/** Mirrors homeProps in cmd/web/main.go. */
type Props = {
  count: number;
};

function Home({ count: initial }: Props) {
  const { appName, user } = useShared();
  const [count, setCount] = useState(initial);

  return (
    <main className="page">
      <h1>Hello, {user}</h1>
      <p className="muted">
        Esta página inteira é React. O {appName} entregou o shell e o estado
        inicial; o <code>&lt;main&gt;</code> é todo daqui para baixo.
      </p>

      <div className="counter">
        <button onClick={() => setCount(count - 1)}>−</button>
        <output>{count}</output>
        <button onClick={() => setCount(count + 1)}>+</button>
      </div>

      <p className="muted">
        O valor inicial <strong>{initial}</strong> veio do handler Go.
      </p>
    </main>
  );
}

mount(Home);
