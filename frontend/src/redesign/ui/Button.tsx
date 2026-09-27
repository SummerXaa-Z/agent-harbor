import type { ReactNode } from "react";

export type ButtonVariant = "primary" | "success" | "ghost" | "danger-ghost" | "link";

interface ButtonBaseProps {
  "aria-label"?: string;
  children: ReactNode;
  disabled?: boolean;
  icon?: ReactNode;
  size?: "md" | "sm";
  title?: string;
  variant?: ButtonVariant;
}

// Every button must do something: run a handler, navigate, or submit its form.
// The union makes an action-less button a type error (no zombie buttons).
type ButtonAction =
  | { href?: never; onClick: () => void; type?: "button" }
  | { href: string; onClick?: never; type?: never }
  | { href?: never; onClick?: never; type: "submit" };

export type ButtonProps = ButtonBaseProps & ButtonAction;

export function buttonClassName(variant: ButtonVariant = "primary", size: "md" | "sm" = "md"): string {
  if (variant === "link") return "btn-link";
  return `btn btn-${variant}${size === "sm" ? " btn-sm" : ""}`;
}

export function Button(props: ButtonProps) {
  const className = buttonClassName(props.variant, props.size);
  const content = (
    <>
      {props.icon}
      {props.children}
    </>
  );
  if (props.href !== undefined) {
    return (
      <a aria-label={props["aria-label"]} className={className} href={props.href} title={props.title}>
        {content}
      </a>
    );
  }
  return (
    <button
      aria-label={props["aria-label"]}
      className={className}
      disabled={props.disabled}
      onClick={props.onClick}
      title={props.title}
      type={props.type === "submit" ? "submit" : "button"}
    >
      {content}
    </button>
  );
}
