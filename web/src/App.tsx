import { Navigate, Route, Routes } from "react-router-dom";
import Today from "./pages/Today";
import Match from "./pages/Match";
import Results from "./pages/Results";

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Today />} />
      <Route path="/results" element={<Results />} />
      <Route path="/experts" element={<Navigate to="/results" replace />} />
      <Route path="/matches/:id" element={<Match />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
