import { Navigate, Route, Routes } from "react-router-dom";
import { Layout } from "./components/Layout";
import { useAuth } from "./context/AuthContext";
import { AuthCallback } from "./pages/AuthCallback";
import { Admin } from "./pages/Admin";
import { Families } from "./pages/Families";
import { FamilyView } from "./pages/FamilyView";
import { Login } from "./pages/Login";

function Protected({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth();
  if (loading) return <div className="py-20 text-center text-slate-500">Loading...</div>;
  if (!user) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

export default function App() {
  return (
    <Routes>
      <Route path="/auth/callback" element={<AuthCallback />} />
      <Route
        path="/login"
        element={
          <Layout>
            <Login />
          </Layout>
        }
      />
      <Route
        path="/"
        element={
          <Protected>
            <Layout>
              <Families />
            </Layout>
          </Protected>
        }
      />
      <Route
        path="/admin"
        element={
          <Protected>
            <Layout>
              <Admin />
            </Layout>
          </Protected>
        }
      />
      <Route
        path="/families/:familyId"
        element={
          <Protected>
            <Layout>
              <FamilyView />
            </Layout>
          </Protected>
        }
      />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}