const rawAppURL = (import.meta.env.VITE_APP_URL as string | undefined)?.trim() || "";
const normalizedAppURL = rawAppURL.replace(/\/+$/, "");

export function appHref(path: string): string {
  if (!normalizedAppURL) {
    return path;
  }
  return `${normalizedAppURL}${path}`;
}
