import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import {themeHintKey} from './components/themeOptions'
import {normalizeTheme} from './services/mockBackend'
import './styles.css'

// Paint the configured theme before the first React render so secondary
// windows (and relaunches) do not flash the default dark theme.
const urlTheme = new URLSearchParams(window.location.search).get('theme')
let hintedTheme: string | null = urlTheme
if (!hintedTheme) {
  try {
    hintedTheme = window.localStorage.getItem(themeHintKey)
  } catch {
    hintedTheme = null
  }
}
if (hintedTheme) {
  document.documentElement.dataset.theme = normalizeTheme(hintedTheme)
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
