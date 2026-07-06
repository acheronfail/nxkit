import './index.css';
import { mount } from 'svelte';
import App from './App.svelte';

const target = document.getElementById('app');
if (target) {
  mount(App, { target });
} else {
  alert('Failed to mount app - no DOM node found with id "app"');
}
