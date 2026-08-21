import { Navigate, Route, Routes } from "react-router-dom";
import Today from "./pages/Today";
import Match from "./pages/Match";
import Results from "./pages/Results";
import Week from "./pages/Week";
import SFC from "./pages/SFC";
import SFCDetail from "./pages/SFCDetail";
import AccessGate from "./pages/AccessGate";
import AdminAccess from "./pages/AdminAccess";

export default function App() {
  return (
    <Routes>
      <Route path={import.meta.env.VITE_ADMIN_PATH || "/console-k7m4x9"} element={<AdminAccess />} />
      <Route path="/" element={<AccessGate><Today /></AccessGate>} />
      <Route path="/sfc" element={<AccessGate><SFC /></AccessGate>} />
      <Route path="/sfc/:no" element={<AccessGate><SFCDetail /></AccessGate>} />
      <Route path="/results" element={<AccessGate><Results /></AccessGate>} />
      <Route path="/week" element={<AccessGate><Week /></AccessGate>} />
      <Route path="/experts" element={<Navigate to="/results" replace />} />
      <Route path="/matches/:id" element={<AccessGate><Match /></AccessGate>} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
