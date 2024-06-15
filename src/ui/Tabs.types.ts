import { Component } from "svelte";

export interface Tab {
  id: string;
  displayName: string;
  component: Component;
}
