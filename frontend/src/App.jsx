import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState, useEffect } from 'react'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import InstanceDetail from './pages/InstanceDetail'
import Deploy from './pages/Deploy'
import Users from './pages/Users'
import AdminIdentityModal from './components/AdminIdentityModal'

const qc = new QueryClient({
  defaultOptions: { queries: { retry: 1, staleTime: 5000 } },
})

export default function App() {
  const [hasIdentity, setHasIdentity] = useState(!!localStorage.getItem('adminEmail'))

  useEffect(() => {
    const check = () => setHasIdentity(!!localStorage.getItem('adminEmail'))
    window.addEventListener('storage', check)
    return () => window.removeEventListener('storage', check)
  }, [])

  return (
    <QueryClientProvider client={qc}>
      {!hasIdentity && <AdminIdentityModal onSave={() => setHasIdentity(true)} />}
      <BrowserRouter>
        <Routes>
          <Route element={<Layout />}>
            <Route path="/" element={<Dashboard />} />
            <Route path="/instances/:deploymentId" element={<InstanceDetail />} />
            <Route path="/deploy" element={<Deploy />} />
            <Route path="/users" element={<Users />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
