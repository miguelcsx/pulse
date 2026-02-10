import { forwardRef } from "react";
import type { ButtonHTMLAttributes, ReactNode } from "react";
import Spinner from "./Spinner";

const variantClasses = {
  primary: "bg-indigo-600 hover:bg-indigo-500 text-white",
  secondary:
    "bg-[var(--color-surface)] border border-[var(--color-border)] text-[var(--color-text)] hover:bg-[var(--color-surface-hover)]",
  ghost: "bg-transparent hover:bg-[var(--color-surface)] text-[var(--color-text)]",
  danger: "bg-red-600 hover:bg-red-500 text-white",
} as const;

const sizeClasses = {
  sm: "px-3 py-1.5 text-sm",
  md: "px-4 py-2 text-sm",
  lg: "px-6 py-3 text-base",
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
        className={`inline-flex items-center justify-center rounded-lg font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 focus:ring-offset-[var(--color-bg)] disabled:opacity-50 disabled:pointer-events-none ${variantClasses[variant]} ${sizeClasses[size]} ${className}`}
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
