import { Routes, Route, Navigate } from "react-router-dom";
import { useAuth } from "./auth/AuthContext";
import Layout from "./components/Layout";
import Login from "./pages/Login";
import Landing from "./pages/Landing";
import Dashboard from "./pages/Dashboard";
import Transactions from "./pages/Transactions";
import Stores from "./pages/Stores";
import StoreDetail from "./pages/StoreDetail";
import Provider from "./pages/Provider";
import Docs from "./pages/Docs";
import Settings from "./pages/Settings";

function RequireAuth() {
  const { isAuthenticated } = useAuth();
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return <Layout />;
}

export default function App() {
  const { isAuthenticated, loading } = useAuth();

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-mint border-t-transparent" />
      </div>
    );
  }

  return (
    <Routes>
      {/* Public */}
      <Route path="/" element={<Landing />} />
      <Route
        path="/login"
        element={isAuthenticated ? <Navigate to="/panel" replace /> : <Login />}
      />

      {/* Protected panel */}
      <Route element={<RequireAuth />}>
        <Route path="/panel" element={<Dashboard />} />
        <Route path="/panel/transactions" element={<Transactions />} />
        <Route path="/panel/stores" element={<Stores />} />
        <Route path="/panel/stores/:id" element={<StoreDetail />} />
        <Route path="/panel/provider" element={<Provider />} />
        <Route path="/panel/docs" element={<Docs />} />
        <Route path="/panel/settings" element={<Settings />} />
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}