import type { ReactNode } from "react";

export interface KvItem {
  key: string;
  label: ReactNode;
  value: ReactNode;
}

export function KvList({ items }: { items: readonly KvItem[] }) {
  return (
    <dl className="kv">
      {items.map((item) => (
        <div key={item.key}>
          <dt className="kv-k">{item.label}</dt>
          <dd className="kv-v">{item.value}</dd>
        </div>
      ))}
    </dl>
  );
}
