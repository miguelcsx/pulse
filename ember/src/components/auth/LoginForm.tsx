import { useState } from "react";
import type { FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import type { AxiosError } from "axios";
import { demoLogin, login } from "../../api/auth";
import { useAuthStore } from "../../store/authStore";
import Input from "../ui/Input";
import Button from "../ui/Button";

function randomDemoHandle(): string {
  const suffix = Math.random().toString(36).slice(2, 7);
  return `guest_${suffix}`;
}

export default function LoginForm() {
  const [handle, setHandle] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [demoLoading, setDemoLoading] = useState(false);
  const [showSignIn, setShowSignIn] = useState(false);

  const setTokens = useAuthStore((s) => s.setTokens);
  const setUser = useAuthStore((s) => s.setUser);
  const navigate = useNavigate();

  async function startDemo(demoHandle: string) {
    setError("");
    setDemoLoading(true);
    try {
      const res = await demoLogin(demoHandle);
      setTokens(res.access_token);
      setUser(res.user);
      navigate("/");
    } catch (err) {
      const axiosErr = err as AxiosError<{ error: string }>;
      setError(
        axiosErr.response?.data?.error ?? "Could not start a demo session",
      );
    } finally {
      setDemoLoading(false);
    }
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const res = await login(email, password);
      setTokens(res.access_token);
      setUser(res.user);
      navigate("/");
    } catch (err) {
      const axiosErr = err as AxiosError<{ error: string }>;
      setError(axiosErr.response?.data?.error ?? "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex flex-col gap-5">
      {/* Primary path: try instantly, no account needed */}
      <div className="flex flex-col gap-3">
        <Button
          type="button"
          variant="accent"
          size="lg"
          loading={demoLoading}
          onClick={() => startDemo(randomDemoHandle())}
        >
          Try Pulse instantly
        </Button>
        <p className="text-center text-xs text-[var(--color-text-muted)]">
          No sign-up — you get a throwaway demo account to explore everything.
        </p>

        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (handle.trim()) startDemo(handle.trim());
          }}
          className="flex items-end gap-2"
        >
          <div className="flex-1">
            <Input
              label="Or pick a demo name"
              value={handle}
              onChange={(e) => setHandle(e.target.value)}
              placeholder="yourname"
              minLength={3}
              maxLength={30}
              pattern="[A-Za-z0-9_.\-]+"
            />
          </div>
          <Button
            type="submit"
            variant="secondary"
            disabled={demoLoading || handle.trim().length < 3}
          >
            Enter
          </Button>
        </form>
      </div>

      {error && <p className="text-sm text-[var(--color-error)]">{error}</p>}

      {/* Secondary: real account sign-in, tucked away to reduce clutter */}
      {showSignIn ? (
        <form
          onSubmit={handleSubmit}
          className="flex flex-col gap-4 border-t border-[var(--color-border)] pt-5"
        >
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
          <Button type="submit" loading={loading}>
            Sign in
          </Button>
          <p className="text-center text-sm text-[var(--color-text-muted)]">
            Don&apos;t have an account?{" "}
            <Link
              to="/register"
              className="text-[var(--color-primary)] hover:opacity-80"
            >
              Register
            </Link>
          </p>
        </form>
      ) : (
        <div className="flex items-center gap-3 border-t border-[var(--color-border)] pt-4 text-xs text-[var(--color-text-muted)]">
          <span>Have an account?</span>
          <button
            type="button"
            onClick={() => setShowSignIn(true)}
            className="font-medium text-[var(--color-text)] hover:opacity-80"
          >
            Sign in
          </button>
        </div>
      )}
    </div>
  );
}
