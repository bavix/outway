import { render } from 'preact';
import { App } from './routes.js';
import '../styles/main.css';

render(<App />, document.getElementById('app')!);
