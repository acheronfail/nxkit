import { expect, test } from 'vitest';
import { render } from '@testing-library/svelte';
import App from './App.svelte';

test('it works', () => {
  const screen = render(App, {});
  expect(screen.getByTestId('nro-forwarder')).toBeInTheDocument();
});
