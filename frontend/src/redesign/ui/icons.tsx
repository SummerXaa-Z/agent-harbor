import { CircleCheck, CircleX, Info, TriangleAlert } from "lucide-react";
import type { ReactNode } from "react";
import type { RedesignView } from "../router";

export type Tone = "success" | "warning" | "danger" | "info" | "neutral";

interface IconProps {
  className?: string;
  size?: number;
}

function Svg({ children, className, size = 17 }: IconProps & { children: ReactNode }) {
  return (
    <svg aria-hidden="true" className={className} fill="none" focusable="false" height={size} viewBox="0 0 24 24" width={size}>
      {children}
    </svg>
  );
}

function Stroke({ d, width = 1.7 }: { d: string; width?: number }) {
  return <path d={d} stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth={width} />;
}

export function LogoIcon({ className, size = 18 }: IconProps) {
  return (
    <Svg className={className} size={size}>
      <Stroke d="M7 6h10M7 12h7M7 18h10" width={2.2} />
    </Svg>
  );
}

export function UserSurfaceIcon({ className, size = 24 }: IconProps) {
  return (
    <Svg className={className} size={size}>
      <rect height="12" rx="2.5" stroke="currentColor" strokeWidth={1.8} width="17" x="3.5" y="4.5" />
      <Stroke d="M8 20h8M12 16.5V20" width={1.8} />
    </Svg>
  );
}

export function AdminSurfaceIcon({ className, size = 24 }: IconProps) {
  return (
    <Svg className={className} size={size}>
      <Stroke d="M12 3l7.5 3v5c0 4.6-3.1 8.2-7.5 10-4.4-1.8-7.5-5.4-7.5-10V6L12 3z" width={1.8} />
      <Stroke d="M9 12l2 2 4-4" width={1.8} />
    </Svg>
  );
}

export function ListCheckIcon({ className, size = 14 }: IconProps) {
  return (
    <Svg className={className} size={size}>
      <Stroke d="M5 12.5l4 4L19 7" width={2.2} />
    </Svg>
  );
}

export function GoArrowIcon({ className, size = 16 }: IconProps) {
  return (
    <Svg className={className} size={size}>
      <Stroke d="M5 12h14M13 6l6 6-6 6" width={2} />
    </Svg>
  );
}

export function SurfaceSwitchIcon({ className, size = 15 }: IconProps) {
  return (
    <Svg className={className} size={size}>
      <Stroke d="M8 7l-5 5 5 5M16 7l5 5-5 5M13.5 4l-3 16" />
    </Svg>
  );
}

export function RefreshIcon({ className, size = 17 }: IconProps) {
  return (
    <Svg className={className} size={size}>
      <Stroke d="M20 11A8 8 0 006.3 6.3L4 8.5M4 4v4.5h4.5M4 13a8 8 0 0013.7 4.7L20 15.5M20 20v-4.5h-4.5" />
    </Svg>
  );
}

export function CloseIcon({ className, size = 16 }: IconProps) {
  return (
    <Svg className={className} size={size}>
      <Stroke d="M6 6l12 12M18 6L6 18" width={2} />
    </Svg>
  );
}

const navIconPaths: Record<RedesignView, ReactNode> = {
  home: <Stroke d="M4 10.5L12 4l8 6.5V20a1 1 0 01-1 1h-4.5v-6h-5v6H5a1 1 0 01-1-1v-9.5z" />,
  ask: (
    <>
      <circle cx="11" cy="11" r="6.5" stroke="currentColor" strokeWidth={1.7} />
      <Stroke d="M16 16l4.5 4.5" />
    </>
  ),
  mine: (
    <>
      <Stroke d="M12 3l7 3v5c0 4.3-2.9 7.6-7 9-4.1-1.4-7-4.7-7-9V6l7-3z" />
      <Stroke d="M9.5 12l1.8 1.8 3.4-3.6" />
    </>
  ),
  apply: (
    <>
      <rect height="16" rx="3" stroke="currentColor" strokeWidth={1.7} width="16" x="4" y="4" />
      <Stroke d="M9 11.5l2 2 4-4.5" />
    </>
  ),
  golive: <Stroke d="M12 20V11M7 13l5-8 5 8M5 20h14" />,
  cockpit: (
    <>
      <circle cx="12" cy="12" r="8.5" stroke="currentColor" strokeWidth={1.7} />
      <Stroke d="M12 8v4M12 16h.01" width={1.9} />
    </>
  ),
  approvals: (
    <>
      <rect height="17" rx="2.5" stroke="currentColor" strokeWidth={1.7} width="16" x="4" y="3.5" />
      <Stroke d="M8.5 8h7M8.5 12h7M8.5 16h4" width={1.6} />
    </>
  ),
  traces: (
    <>
      <Stroke d="M4 18l4.5-5 3.5 3 5-7L20 6" />
      <Stroke d="M4 21h16" width={1.5} />
    </>
  ),
  tenants: (
    <>
      <Stroke d="M4 8l8-4 8 4-8 4-8-4z" />
      <Stroke d="M4 12l8 4 8-4M4 16l8 4 8-4" width={1.5} />
    </>
  ),
  registry: (
    <>
      <rect height="14" rx="2.5" stroke="currentColor" strokeWidth={1.7} width="16" x="4" y="5" />
      <Stroke d="M4 10h16M9 5v14" width={1.6} />
    </>
  ),
  capabilities: (
    <>
      <Stroke d="M13 3l7 7-9 9H4v-7l9-9z" />
      <Stroke d="M11 10l3 3" width={1.6} />
    </>
  ),
  policies: <Stroke d="M12 3l7 3v5c0 4.6-3.1 8.2-7.5 10C7.1 19.2 4 15.6 4 11V6l8-3z" />,
  routes: (
    <>
      <circle cx="5.5" cy="12" r="2.5" stroke="currentColor" strokeWidth={1.7} />
      <circle cx="18.5" cy="6" r="2.5" stroke="currentColor" strokeWidth={1.7} />
      <circle cx="18.5" cy="18" r="2.5" stroke="currentColor" strokeWidth={1.7} />
      <Stroke d="M8 11l8-4M8 13l8 4" width={1.6} />
    </>
  ),
  admin: (
    <>
      <circle cx="12" cy="12" r="3" stroke="currentColor" strokeWidth={1.7} />
      <Stroke
        d="M12 3v2.5M12 18.5V21M3 12h2.5M18.5 12H21M5.6 5.6l1.8 1.8M16.6 16.6l1.8 1.8M18.4 5.6l-1.8 1.8M7.4 16.6l-1.8 1.8"
        width={1.5}
      />
    </>
  ),
};

export function NavIcon({ className, size = 17, view }: IconProps & { view: RedesignView }) {
  return (
    <Svg className={className} size={size}>
      {navIconPaths[view]}
    </Svg>
  );
}

// Status is always icon + shape + text, never color alone.
export function ToneIcon({ size = 16, tone }: { size?: number; tone: Tone }) {
  if (tone === "success") return <CircleCheck aria-hidden="true" size={size} />;
  if (tone === "warning") return <TriangleAlert aria-hidden="true" size={size} />;
  if (tone === "danger") return <CircleX aria-hidden="true" size={size} />;
  return <Info aria-hidden="true" size={size} />;
}
