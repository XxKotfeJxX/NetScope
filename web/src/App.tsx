import { Route, Routes } from "react-router-dom";
import { Layout } from "./components/Layout";
import { DashboardPage } from "./pages/DashboardPage";
import { CompareRunsPage } from "./pages/CompareRunsPage";
import { HistoryPage } from "./pages/HistoryPage";
import { MonitoringPage } from "./pages/MonitoringPage";
import { RunPage } from "./pages/RunPage";
import { SettingsPage } from "./pages/SettingsPage";
import { TargetDetailPage } from "./pages/TargetDetailPage";
import { TargetsPage } from "./pages/TargetsPage";

export function App() {
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
      </Route>
    </Routes>
  );
}
