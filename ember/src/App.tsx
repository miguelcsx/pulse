import { lazy, Suspense, useEffect, useState } from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import { useAuthStore } from "./store/authStore";
import client from "./api/client";
import { getMe } from "./api/auth";
import { startEventLoop, stopEventLoop } from "./api/events";
import AppShell from "./components/layout/AppShell";
import ProtectedRoute from "./components/auth/ProtectedRoute";
import Spinner from "./components/ui/Spinner";

const Login = lazy(() => import("./pages/Login"));
const Register = lazy(() => import("./pages/Register"));
const Feed = lazy(() => import("./pages/Feed"));
const Upload = lazy(() => import("./pages/Upload"));
const Suggestions = lazy(() => import("./pages/Suggestions"));
const Rooms = lazy(() => import("./pages/Rooms"));
const RoomView = lazy(() => import("./pages/RoomView"));
const Paths = lazy(() => import("./pages/Paths"));
const PathView = lazy(() => import("./pages/PathView"));
const Profile = lazy(() => import("./pages/Profile"));
const Settings = lazy(() => import("./pages/Settings"));
const NotFound = lazy(() => import("./pages/NotFound"));

export default function App() {
  const isAuthenticated = useAuthStore((s) => !!s.accessToken);
  const setTokens = useAuthStore((s) => s.setTokens);
  const setUser = useAuthStore((s) => s.setUser);
  const logout = useAuthStore((s) => s.logout);
  const [bootstrapped, setBootstrapped] = useState(false);

  useEffect(() => {
    let mounted = true;

    async function bootstrapAuth() {
      if (isAuthenticated) {
        if (mounted) setBootstrapped(true);
        return;
      }

      try {
        const refresh = await client.post<{ access_token: string }>(
          "/auth/refresh",
          {},
        );
        if (!mounted) return;
        setTokens(refresh.data.access_token);
        try {
          const me = await getMe();
          if (mounted) {
            setUser(me);
          }
        } catch {
          // Access token is still valid even if profile fetch fails transiently.
        }
      } catch {
        if (mounted) {
          logout();
        }
      } finally {
        if (mounted) {
          setBootstrapped(true);
        }
      }
    }

    bootstrapAuth();
    return () => {
      mounted = false;
    };
  }, [isAuthenticated, logout, setTokens, setUser]);

  // Start/stop the event tracking loop based on auth state
  useEffect(() => {
    if (isAuthenticated) {
      startEventLoop();
    } else {
      stopEventLoop();
    }

    return () => {
      stopEventLoop();
    };
  }, [isAuthenticated]);

  if (!bootstrapped) {
    return <div className="min-h-dvh bg-[var(--color-bg)]" />;
  }

  return (
    <Suspense
      fallback={
        <div className="min-h-dvh flex items-center justify-center bg-[var(--color-bg)]">
          <Spinner size="lg" />
        </div>
      }
    >
      <Routes>
        <Route
          path="/login"
          element={isAuthenticated ? <Navigate to="/" /> : <Login />}
        />
        <Route
          path="/register"
          element={isAuthenticated ? <Navigate to="/" /> : <Register />}
        />
        <Route element={<ProtectedRoute />}>
          <Route element={<AppShell />}>
            <Route path="/" element={<Feed />} />
            <Route path="/upload" element={<Upload />} />
            <Route path="/suggestions" element={<Suggestions />} />
            <Route path="/rooms" element={<Rooms />} />
            <Route path="/rooms/:id" element={<RoomView />} />
            <Route path="/paths" element={<Paths />} />
            <Route path="/paths/new" element={<PathView />} />
            <Route path="/paths/:id" element={<PathView />} />
            <Route path="/profile/:id" element={<Profile />} />
            <Route path="/settings" element={<Settings />} />
          </Route>
        </Route>
        <Route path="*" element={<NotFound />} />
      </Routes>
    </Suspense>
  );
}
