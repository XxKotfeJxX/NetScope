import { Route, Routes, useLocation } from "react-router-dom";
import { Layout } from "./components/Layout";
import { useAuth } from "./auth/AuthContext";
import { DashboardPage } from "./pages/DashboardPage";
import { CompareRunsPage } from "./pages/CompareRunsPage";
import { HistoryPage } from "./pages/HistoryPage";
import { MonitoringPage } from "./pages/MonitoringPage";
import { RunPage } from "./pages/RunPage";
import { SettingsPage } from "./pages/SettingsPage";
import { TargetDetailPage } from "./pages/TargetDetailPage";
import { TargetsPage } from "./pages/TargetsPage";
import { AuthPage } from "./pages/AuthPage";
import { WorkspacePage } from "./pages/WorkspacePage";
import { PublicReportPage } from "./pages/PublicReportPage";

export function App() {
  const { account, loading } = useAuth();
  const location = useLocation();

  if (location.pathname.startsWith("/shared/")) {
    return (
      <Routes>
        <Route path="/shared/:token" element={<PublicReportPage />} />
      </Routes>
    );
  }

  if (loading) {
    return (
      <main className="auth-loading">
        <span className="brand">
          NETSCOPE<span aria-hidden="true">/</span>
        </span>
        <p>Resolving workspace…</p>
      </main>
    );
  }
  if (!account) {
    return <AuthPage />;
  }

  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<DashboardPage />} />
        <Route path="/runs/compare" element={<CompareRunsPage />} />
        <Route path="/runs/:id" element={<RunPage />} />
        <Route path="/targets" element={<TargetsPage />} />
        <Route path="/targets/:id" element={<TargetDetailPage />} />
        <Route path="/monitoring" element={<MonitoringPage />} />
        <Route path="/history" element={<HistoryPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="/workspace" element={<WorkspacePage />} />
      </Route>
    </Routes>
  );
}
