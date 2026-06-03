import axios from "axios";
import { API_BASE } from "@pulse/drift/constants";
import { useAuthStore } from "../store/authStore";

const configuredAPIBase = import.meta.env.VITE_API_BASE || API_BASE;
const configuredCSRFCookieName = import.meta.env.VITE_CSRF_COOKIE_NAME || "pulse_csrf_token";
const configuredCSRFHeaderName = import.meta.env.VITE_CSRF_HEADER_NAME || "X-CSRF-Token";

const client = axios.create({
  baseURL: configuredAPIBase,
  timeout: 15000,
  withCredentials: true,
});

client.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }

  const method = String(config.method ?? "get").toLowerCase();
  if (method !== "get" && method !== "head" && method !== "options") {
    const csrfToken = getCookie(configuredCSRFCookieName);
    if (csrfToken) {
      config.headers[configuredCSRFHeaderName] = csrfToken;
    }
  }

  return config;
});

let refreshPromise: Promise<string> | null = null;

client.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;
    const requestURL = String(originalRequest?.url ?? "");

    if (
      error.response?.status !== 401 ||
      !originalRequest ||
      originalRequest._retry ||
      requestURL.includes("/auth/refresh") ||
      requestURL.includes("/auth/demo") ||
      requestURL.includes("/auth/login") ||
      requestURL.includes("/auth/register") ||
      requestURL.includes("/auth/logout")
    ) {
      return Promise.reject(error);
    }

    originalRequest._retry = true;

    const { setTokens, logout } = useAuthStore.getState();

    try {
      // Deduplicate concurrent refresh attempts
      if (!refreshPromise) {
        refreshPromise = axios
          .post<{ access_token: string }>(
            `${configuredAPIBase}/auth/refresh`,
            {},
            { withCredentials: true },
          )
          .then((res) => {
            setTokens(res.data.access_token);
            return res.data.access_token;
          })
          .finally(() => {
            refreshPromise = null;
          });
      }

      const newAccessToken = await refreshPromise;
      originalRequest.headers.Authorization = `Bearer ${newAccessToken}`;
      return client(originalRequest);
    } catch {
      logout();
      window.location.href = "/login";
      return Promise.reject(error);
    }
  },
);

export default client;

function getCookie(name: string): string | null {
  if (typeof document === "undefined" || !name) {
    return null;
  }
  const parts = document.cookie.split("; ");
  for (const part of parts) {
    const [k, ...rest] = part.split("=");
    if (k === name) {
      return decodeURIComponent(rest.join("="));
    }
  }
  return null;
}
