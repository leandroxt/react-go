import { LitElement, html } from 'lit';

/** The shape the Go handler marshals into the element's JSON script tag. */
interface CounterState {
  count: number;
  step: number;
}

export class AppCounter extends LitElement {
  static properties = {
    count: { type: Number },
    step: { type: Number },
  };

  // `declare` emits nothing, so these are immune to whichever class-field
  // semantics are in effect. Initial values are assigned in the constructor.
  declare count: number;
  declare step: number;

  constructor() {
    super();
    this.count = 0;
    this.step = 1;
  }

  // Light DOM, so main.css applies directly.
  createRenderRoot() {
    return this;
  }

  connectedCallback() {
    // Read server state first: Lit's first render replaces the render root's
    // children and would eat the script tag.
    this.#readInitialState();
    super.connectedCallback();
  }

  #readInitialState() {
    const script = this.querySelector<HTMLScriptElement>(
      'script[type="application/json"]',
    );
    if (!script?.textContent) return;

    try {
      const state = JSON.parse(script.textContent) as Partial<CounterState>;
      if (typeof state.count === 'number') this.count = state.count;
      if (typeof state.step === 'number') this.step = state.step;
    } catch (err) {
      console.error('app-counter: could not parse initial state', err);
    }
  }

  render() {
    return html`
      <div class="counter">
        <button
          class="counter__button"
          aria-label="Decrement"
          @click=${() => this.#bump(-this.step)}
        >
          −
        </button>
        <output class="counter__value">${this.count}</output>
        <button
          class="counter__button"
          aria-label="Increment"
          @click=${() => this.#bump(this.step)}
        >
          +
        </button>
      </div>
    `;
  }

  #bump(delta: number) {
    this.count += delta;
  }
}

customElements.define('app-counter', AppCounter);

declare global {
  interface HTMLElementTagNameMap {
    'app-counter': AppCounter;
  }
}
