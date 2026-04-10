import { Routes, Route, Navigate } from 'react-router-dom'
import Layout from './components/Layout'
import Scenes from './pages/Scenes'
import Actors from './pages/Actors'
import Files from './pages/Files'
import Settings from './pages/Settings'
import Logs from './pages/Logs'

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Navigate to="scenes" replace />} />
        <Route path="scenes" element={<Scenes />} />
        <Route path="actors" element={<Actors />} />
        <Route path="files" element={<Files />} />
        <Route path="settings" element={<Settings />} />
        <Route path="logs" element={<Logs />} />
      </Route>
    </Routes>
  )
}
