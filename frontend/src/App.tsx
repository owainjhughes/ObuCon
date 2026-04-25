import React from "react";
import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import { AuthProvider } from "./auth/AuthContext";
import ProtectedRoute from "./auth/ProtectedRoute";
import Home from "./pages/Home";
import Login from "./pages/Login";
import Register from "./pages/Register";
import Profile from "./pages/Profile";
import Vocab from "./pages/Vocab";
import Dictionary from "./pages/Dictionary";
import Analysis from "./pages/Analysis";

function protectedElement(element: React.ReactElement) {
  return <ProtectedRoute>{element}</ProtectedRoute>;
}

function App() {
  return (
    <AuthProvider>
      <Router>
        <div className="App">
          <Routes>
            <Route path="/" element={protectedElement(<Home />)} />
            <Route path="/vocabulary" element={protectedElement(<Vocab />)} />
            <Route path="/dictionary" element={protectedElement(<Dictionary />)} />
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />
            <Route path="/profile" element={<Profile />} />
            <Route path="/analysis" element={protectedElement(<Analysis />)} />
          </Routes>
        </div>
      </Router>
    </AuthProvider>
  );
}

export default App;