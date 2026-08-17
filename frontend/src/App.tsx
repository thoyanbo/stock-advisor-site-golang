import { Routes, Route } from "react-router-dom";
import HomePage from "./pages/HomePage";
import DCFPage from "./pages/DCFPage";

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<HomePage />} />
      <Route path="/stock/:ticker/dcf" element={<DCFPage />} />
    </Routes>
  );
}
