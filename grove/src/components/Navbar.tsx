import { useState, useEffect } from "react";
import { appHref } from "../config";

export default function Navbar() {
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const handleScroll = () => setScrolled(window.scrollY > 20);
    window.addEventListener("scroll", handleScroll, { passive: true });
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  return (
    <nav
      className={`fixed top-0 w-full z-50 transition-all duration-300 border-b ${
        scrolled
          ? "bg-[var(--color-bg)]/95 backdrop-blur-md border-[var(--color-border)]"
          : "bg-transparent border-transparent"
      }`}
    >
      <div className="max-w-5xl mx-auto px-6 h-16 flex items-center justify-between">
        <span className="text-xl font-bold bg-gradient-to-r from-[var(--color-primary)] to-pink-500 bg-clip-text text-transparent">
          Pulse
        </span>
        <div className="flex items-center gap-4">
          <a
            href={appHref("/login")}
            className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text)] transition-colors"
          >
            Log in
          </a>
          <a
            href={appHref("/register")}
            className="text-sm px-4 py-2 rounded-lg bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white transition-colors"
          >
            Get Started
          </a>
        </div>
      </div>
    </nav>
  );
}
