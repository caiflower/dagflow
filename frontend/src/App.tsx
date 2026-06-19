import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import Layout from './components/layout/Layout';
import FlowListPage from './pages/FlowListPage';
import FlowEditorPage from './pages/FlowEditorPage';
import ExecutionPage from './pages/ExecutionPage';
import ProtocolPage from './pages/ProtocolPage';

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<Navigate to="/flows" replace />} />
          <Route path="/flows" element={<FlowListPage />} />
          <Route path="/flows/:id" element={<FlowEditorPage />} />
          <Route path="/executions" element={<ExecutionPage />} />
          <Route path="/protocols" element={<ProtocolPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
