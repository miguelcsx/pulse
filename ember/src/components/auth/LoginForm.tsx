import { useState } from "react";
import type { FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import type { AxiosError } from "axios";
import client from "../../api/client";
import { demoLogin } from "../../api/auth";
import { useAuthStore } from "../../store/authStore";
import type { AuthTokens } from "@pulse/drift/types";
import Input from "../ui/Input";
import Button from "../ui/Button";

const demoAuthEnabled = import.meta.env.VITE_DEMO_AUTH_ENABLED === "true";

export default function LoginForm() {
  const [handle, setHandle] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [demoLoading, setDemoLoading] = useState(false);

  const setTokens = useAuthStore((s) => s.setTokens);
  const setUser = useAuthStore((s) => s.setUser);
  const navigate = useNavigate();

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await client.post<AuthTokens>("/auth/login", {
        email,
        password,
      });
      setTokens(res.data.access_token);
      setUser(res.data.user);
      navigate("/");
    } catch (err) {
      const axiosErr = err as AxiosError<{ error: string }>;
      setError(axiosErr.response?.data?.error ?? "Login failed");
    } finally {
      setLoading(false);
    }
  }

  async function handleDemoSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setDemoLoading(true);

    try {
      const res = await demoLogin(handle);
      setTokens(res.access_token);
      setUser(res.user);
      navigate("/");
    } catch (err) {
      const axiosErr = err as AxiosError<{ error: string }>;
      setError(axiosErr.response?.data?.error ?? "Could not start demo session");
    } finally {
      setDemoLoading(false);
    }
  }

  return (
    <div className="flex flex-col gap-5">
      {demoAuthEnabled && (
        <form onSubmit={handleDemoSubmit} className="flex flex-col gap-4">
          <Input
            label="Demo username"
            value={handle}
            onChange={(e) => setHandle(e.target.value)}
            placeholder="yourname"
            minLength={3}
            maxLength={30}
            pattern="[A-Za-z0-9_.-]+"
            required
          />
          <Button type="submit" loading={demoLoading}>
            Enter demo
          </Button>
        </form>
      )}

      {demoAuthEnabled && (
        <div className="flex items-center gap-3 text-xs text-[var(--color-text-muted)]">
          <span className="h-px flex-1 bg-[var(--color-border)]" />
          <span>or sign in normally</span>
          <span className="h-px flex-1 bg-[var(--color-border)]" />
        </div>
      )}

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <Input
        label="Email"
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        placeholder="you@example.com"
        required
      />
      <Input
        label="Password"
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        placeholder="Password"
        required
      />

      {error && (
        <p className="text-sm text-[var(--color-error)]">{error}</p>
      )}

      <Button type="submit" loading={loading}>
        Sign in
      </Button>

      <p className="text-center text-sm text-[var(--color-text-muted)]">
        Don&apos;t have an account?{" "}
        <Link to="/register" className="text-[var(--color-primary)] hover:opacity-80">
          Register
        </Link>
      </p>
      </form>
    </div>
  );
}
