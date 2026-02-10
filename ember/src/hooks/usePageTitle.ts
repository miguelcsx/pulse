import { useEffect } from "react";

const APP_NAME = "Pulse";

/**
 * Sets the document title for the current page.
 * Automatically resets to the base app name on unmount.
 *
 * @param title - Page-specific title segment (e.g. "Feed", "Settings").
 *                Pass `undefined` or empty string to show just "Pulse".
 */
export function usePageTitle(title?: string): void {
  useEffect(() => {
    const prev = document.title;
    document.title = title ? `${title} — ${APP_NAME}` : APP_NAME;

    return () => {
      document.title = prev;
    };
  }, [title]);
}
