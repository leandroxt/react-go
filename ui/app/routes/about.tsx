import { mount } from '../mount';
import { useShared } from '../shared';

/**
 * No props: this route needs no initial state from Go, only the shared
 * context. Its handler leaves page.Props nil.
 */
function About() {
  const { appName, user } = useShared();

  return (
    <main className="page">
      <h1>Sobre</h1>
      <p className="muted">
        Rota servida por outro bundle. O React e o <code>mount()</code> vivem
        num chunk compartilhado, baixado uma vez só.
      </p>
      <p className="muted">
        Contexto compartilhado nesta página: <strong>{appName}</strong> /{' '}
        <strong>{user}</strong> — reinjetado pelo Go a cada navegação.
      </p>
    </main>
  );
}

mount(About);
