import { useState } from "react";
import type { NodeHealthInfo, VlessNode } from "../../services/api";
import { NodeHealthBadge } from "./NodeHealthBadge";

export type SelectedNode = {
  id: number;
  display_name: string;
  address: string;
  port: number;
};

type Props = {
  /** Currently selected nodes in display order */
  selectedNodes: SelectedNode[];
  /** Nodes available to add (from subscription or all nodes) */
  availableNodes: VlessNode[];
  strategy: "round_robin" | "least_ping";
  /** Health data keyed by position index (from GET /tunnels/:id/nodes) */
  health?: Record<number, NodeHealthInfo>;
  /** Called when selection or strategy changes */
  onChange?: (nodeIds: number[], strategy: "round_robin" | "least_ping") => void;
  /** Read-only mode (tunnel detail view) */
  readOnly?: boolean;
};

const STRATEGY_OPTIONS: { value: "round_robin" | "least_ping"; label: string; hint: string }[] = [
  {
    value: "round_robin",
    label: "Round Robin",
    hint: "Трафик распределяется поровну между узлами. Узлы не проверяются автоматически.",
  },
  {
    value: "least_ping",
    label: "Least Ping",
    hint: "Трафик направляется на самый быстрый узел. Автоматические проверки каждые 10 с.",
  },
];

export function NodeSelector({ selectedNodes, availableNodes, strategy, health, onChange, readOnly }: Props) {
  const [showPicker, setShowPicker] = useState(false);
  const [search, setSearch] = useState("");

  const selectedIds = new Set(selectedNodes.map((n) => n.id));
  const filteredAvailable = availableNodes.filter(
    (n) => !selectedIds.has(n.ID) && (
      n.DisplayName.toLowerCase().includes(search.toLowerCase()) ||
      n.Address.toLowerCase().includes(search.toLowerCase())
    )
  );

  function move(index: number, dir: -1 | 1) {
    if (!onChange) return;
    const next = [...selectedNodes];
    const target = index + dir;
    if (target < 0 || target >= next.length) return;
    [next[index], next[target]] = [next[target], next[index]];
    onChange(next.map((n) => n.id), strategy);
  }

  function remove(id: number) {
    if (!onChange) return;
    const next = selectedNodes.filter((n) => n.id !== id);
    onChange(next.map((n) => n.id), strategy);
  }

  function add(node: VlessNode) {
    if (!onChange) return;
    const next = [...selectedNodes, { id: node.ID, display_name: node.DisplayName, address: node.Address, port: node.Port }];
    onChange(next.map((n) => n.id), strategy);
    setShowPicker(false);
    setSearch("");
  }

  function setStrategy(s: "round_robin" | "least_ping") {
    if (!onChange) return;
    onChange(selectedNodes.map((n) => n.id), s);
  }

  return (
    <div className="space-y-3">
      {/* Node list */}
      <div className="space-y-1.5">
        {selectedNodes.length === 0 && (
          <p className="rounded-lg border border-dashed border-slate-700 px-4 py-6 text-center text-sm text-slate-500">
            Узлы не выбраны. Добавьте хотя бы один VLESS-узел.
          </p>
        )}
        {selectedNodes.map((node, i) => (
          <div
            key={node.id}
            className="flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-2"
          >
            {/* Position */}
            <span className="w-5 text-center text-xs text-slate-600">{i + 1}</span>

            {/* Node info */}
            <div className="min-w-0 flex-1">
              <div className="truncate text-sm text-slate-200">{node.display_name}</div>
              <div className="text-xs text-slate-500">
                {node.address}:{node.port}
              </div>
            </div>

            {/* Health */}
            <NodeHealthBadge health={health?.[i] ?? null} />

            {/* Controls */}
            {!readOnly && (
              <div className="flex items-center gap-1">
                <button
                  type="button"
                  onClick={() => move(i, -1)}
                  disabled={i === 0}
                  title="Переместить вверх"
                  className="rounded p-1 text-slate-500 hover:text-slate-300 disabled:opacity-30"
                >
                  ▲
                </button>
                <button
                  type="button"
                  onClick={() => move(i, 1)}
                  disabled={i === selectedNodes.length - 1}
                  title="Переместить вниз"
                  className="rounded p-1 text-slate-500 hover:text-slate-300 disabled:opacity-30"
                >
                  ▼
                </button>
                <button
                  type="button"
                  onClick={() => remove(node.id)}
                  title="Удалить"
                  className="rounded p-1 text-slate-600 hover:text-red-400"
                >
                  ✕
                </button>
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Add node button */}
      {!readOnly && (
        <div className="relative">
          <button
            type="button"
            onClick={() => setShowPicker((v) => !v)}
            className="flex items-center gap-2 rounded-lg border border-dashed border-slate-700 px-4 py-2 text-sm text-slate-400 hover:border-indigo-500 hover:text-indigo-400 transition-colors"
          >
            <span>+</span> Добавить узел
          </button>

          {showPicker && (
            <div className="absolute left-0 top-full z-20 mt-1 w-96 rounded-lg border border-slate-700 bg-slate-900 shadow-xl">
              <div className="border-b border-slate-800 p-2">
                <input
                  autoFocus
                  type="text"
                  placeholder="Поиск по имени или адресу…"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="w-full rounded bg-slate-800 px-3 py-1.5 text-sm text-slate-200 outline-none"
                />
              </div>
              <div className="max-h-60 overflow-y-auto">
                {filteredAvailable.length === 0 && (
                  <p className="px-4 py-3 text-sm text-slate-500">
                    {availableNodes.length === 0 ? "Нет доступных узлов" : "Не найдено"}
                  </p>
                )}
                {filteredAvailable.map((node) => (
                  <button
                    key={node.ID}
                    type="button"
                    onClick={() => add(node)}
                    className="flex w-full flex-col gap-0.5 px-4 py-2.5 text-left hover:bg-slate-800 transition-colors"
                  >
                    <span className="text-sm text-slate-200">{node.DisplayName}</span>
                    <span className="text-xs text-slate-500">
                      {node.Address}:{node.Port}
                    </span>
                  </button>
                ))}
              </div>
              <div className="border-t border-slate-800 p-2">
                <button
                  type="button"
                  onClick={() => { setShowPicker(false); setSearch(""); }}
                  className="w-full rounded py-1 text-xs text-slate-500 hover:text-slate-300"
                >
                  Закрыть
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Strategy selector */}
      {!readOnly && (
        <div className="space-y-1.5">
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Стратегия балансировки</p>
          <div className="flex gap-3">
            {STRATEGY_OPTIONS.map((opt) => (
              <label
                key={opt.value}
                className={`flex flex-1 cursor-pointer flex-col gap-0.5 rounded-lg border p-3 transition-colors ${
                  strategy === opt.value
                    ? "border-indigo-500 bg-indigo-950/40"
                    : "border-slate-700 hover:border-slate-600"
                }`}
              >
                <div className="flex items-center gap-2">
                  <input
                    type="radio"
                    name="balancing_strategy"
                    value={opt.value}
                    checked={strategy === opt.value}
                    onChange={() => setStrategy(opt.value)}
                    className="accent-indigo-500"
                  />
                  <span className="text-sm font-medium text-slate-200">{opt.label}</span>
                </div>
                <p className="pl-5 text-xs text-slate-500">{opt.hint}</p>
              </label>
            ))}
          </div>
        </div>
      )}

      {/* Strategy display (read-only) */}
      {readOnly && (
        <div className="flex items-center gap-2 text-sm text-slate-400">
          <span className="font-medium text-slate-300">Стратегия:</span>
          {STRATEGY_OPTIONS.find((o) => o.value === strategy)?.label ?? strategy}
        </div>
      )}
    </div>
  );
}
