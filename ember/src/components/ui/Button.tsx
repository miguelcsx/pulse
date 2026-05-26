import { forwardRef } from "react";
import type { ButtonHTMLAttributes, ReactNode } from "react";
import Spinner from "./Spinner";

const variantClasses = {
  primary:
    "bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-[var(--color-bg)] shadow-sm",
  accent:
    "bg-[var(--color-accent)] hover:bg-[var(--color-accent-hover)] text-white shadow-sm",
  secondary:
    "bg-[var(--color-surface)] border border-[var(--color-border)] text-[var(--color-text)] hover:bg-[var(--color-surface-hover)]",
  ghost:
    "bg-transparent hover:bg-[var(--color-surface)] text-[var(--color-text)]",
  danger:
    "bg-[var(--color-error)] hover:opacity-90 text-white shadow-sm",
} as const;

const sizeClasses = {
  sm: "px-3 py-1.5 text-[13px]",
  md: "px-4 py-2.5 text-sm",
  lg: "px-6 py-3 text-[15px]",
} as const;

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: keyof typeof variantClasses;
  size?: keyof typeof sizeClasses;
  loading?: boolean;
  children: ReactNode;
}

const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      variant = "primary",
      size = "md",
      loading = false,
      disabled,
      children,
      className = "",
      ...props
    },
    ref,
  ) => {
    return (
      <button
        ref={ref}
        disabled={disabled || loading}
        className={`inline-flex items-center justify-center rounded-[var(--radius-sm)] font-medium transition-all active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-accent)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--color-bg)] disabled:opacity-40 disabled:pointer-events-none ${variantClasses[variant]} ${sizeClasses[size]} ${className}`}
        {...props}
      >
        {loading && <Spinner size="sm" className="mr-2" />}
        {children}
      </button>
    );
  },
);

Button.displayName = "Button";

export default Button;
