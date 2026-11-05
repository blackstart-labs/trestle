import { mount } from "svelte";
import "@fontsource-variable/mona-sans";
import App from "./App.svelte";
import "./app.css";

mount(App, { target: document.getElementById("app")! });
