import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <div style={{ fontFamily: 'sans-serif', padding: '2rem' }}>
      <h1>RichTalk</h1>
      <p>Загружается...</p>
    </div>
  </StrictMode>
)
