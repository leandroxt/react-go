import { ChangeEvent, FormEvent, useState } from 'react';

import { mount } from '../mount';
import { useShared } from '../shared';

/**
 * No props: this route needs no initial state from Go, only the shared
 * context. Its handler leaves page.Props nil.
 */
function About() {
  const { appName, user } = useShared();
  const [value, setValue] = useState('');
  const [list, setList] = useState<string[]>(['item 1']);

  function onChange(e: ChangeEvent<HTMLInputElement>) {
    setValue(() => e.target.value);
  }

  function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();

    setList((prevState) => [...prevState, value]);
    setValue('');
  }

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

      <div>
        <ul>
          {list.map(item => <li key={item}>{item}</li>)}
        </ul>

        <form onSubmit={onSubmit}>
          <input type="text" value={value} onChange={onChange} placeholder="new item for list" />
          <button type="submit">Add</button>
        </form>

      </div>
    </main>
  );
}

mount(About);
