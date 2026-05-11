import type { ReactNode } from "react";

export function Table({
  headers,
  headerClassNames,
  children,
}: {
  headers: ReactNode[];
  /** Optional per-column classes on `<th>` (e.g. responsive visibility). */
  headerClassNames?: (string | undefined)[];
  children: ReactNode;
}) {
  return (
    <div className="overflow-x-auto rounded-xl border border-[#2a2a3f]">
      <table className="min-w-[36rem] w-full border-collapse text-left text-sm">
        <thead className="bg-[#161620] text-xs uppercase tracking-wide text-[#94a3b8]">
          <tr>
            {headers.map((h, i) => (
              <th
                key={i}
                className={["border-b border-[#2a2a3f] px-4 py-3 font-semibold", headerClassNames?.[i]].filter(Boolean).join(" ")}
              >
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="[&_tr:nth-child(even)]:bg-[#12121a]/80">{children}</tbody>
      </table>
    </div>
  );
}
