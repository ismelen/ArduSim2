/* eslint-disable @typescript-eslint/no-explicit-any */
import { useShallow } from "zustand/shallow";
import Button from "../components/button";
import { useLogs } from "../hooks/useLogs";
import { cn } from "../utils/cn";
import { useState, useEffect } from "react";
import { domain } from "../../wailsjs/go/models";

export default function LogsPage() {
  const [loadAll, logPaths, selectLog, selectedIdx] = useLogs(
    useShallow((s) => [s.loadAll, s.logPaths, s.selectLog, s.selectedIdx]),
  );

  useEffect(() => {
    loadAll();
  }, [loadAll]);

  return (
    <div className="h-[calc(100vh-60px)] flex overflow-clip">
      <aside className="bg-cwhite border-r border-border h-full flex flex-col overflow-hidden w-70 select-none shrink-0">
        <header className="bg-gray border-b border-border px-3 py-1.5 flex items-center justify-between">
          <h4>Logs</h4>
          <Button icon="refresh" onClick={loadAll} />
        </header>
        <main className="p-3 flex-1 flex flex-col gap-2 overflow-y-auto">
          {logPaths.map((e, i) => (
            <LogPathCard
              key={e[0]}
              i={i}
              name={e[0]}
              selected={selectedIdx === i}
              onClick={() => selectLog(selectedIdx === i ? undefined : i)}
            />
          ))}
          {logPaths.length === 0 && (
            <p className="text-sm text-gray-500 italic px-2">No logs found.</p>
          )}
        </main>
      </aside>
      <main className="flex-1 flex flex-col overflow-hidden bg-zinc-950 text-gray-300 relative">
        <FilterBar />
        <LogConsole />
      </main>
    </div>
  );
}

function LogPathCard({
  i,
  name,
  selected,
  onClick,
}: {
  i: number;
  name: string;
  selected: boolean;
  onClick(): void;
}) {
  return (
    <div
      onClick={onClick}
      className={cn(
        `hoverable-gray text-cblack flex items-center gap-2 px-3 py-1.5 cursor-pointer rounded-md`,
        {
          "bg-primary text-white hover:bg-primary": selected,
        },
      )}
    >
      <span className="material-symbols-rounded">folder_zip</span>
      <span className="truncate">{name}</span>
    </div>
  );
}

function FilterBar() {
  const [filter, setFilter, search, isLoading] = useLogs(
    useShallow((s) => [s.filter, s.setFilter, s.search, s.isLoading]),
  );

  const [localFilter, setLocalFilter] = useState<domain.LogFilter>(filter);

  // Sync local if global changes
  useEffect(() => {
    setLocalFilter(filter);
  }, [filter]);

  const handleSearch = () => {
    setFilter(localFilter);
    search();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSearch();
    }
  };

  const updateField = (field: keyof domain.LogFilter, val: string) => {
    setLocalFilter((prev) => {
      const next = new domain.LogFilter(prev);
      (next as any)[field] = val;
      return next;
    });
  };

  return (
    <div className="bg-zinc-900 border-b border-zinc-800 p-2 flex items-center gap-3 text-sm shrink-0 flex-wrap">
      <div className="flex items-center gap-2">
        <span className="text-zinc-400">Level:</span>
        <select
          value={localFilter.Level}
          onChange={(e) => updateField("Level", e.target.value)}
          className="bg-zinc-800 text-zinc-200 border border-zinc-700 rounded px-2 py-1 outline-none focus:border-primary"
        >
          <option value="">ALL</option>
          <option value="INFO">INFO</option>
          <option value="WARN">WARN</option>
          <option value="ERROR">ERROR</option>
          <option value="DEBUG">DEBUG</option>
        </select>
      </div>

      <div className="flex items-center gap-2">
        <span className="text-zinc-400">Instance:</span>
        <input
          type="text"
          value={localFilter.InstanceID}
          onChange={(e) => updateField("InstanceID", e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="e.g. netsim_1"
          className="bg-zinc-800 text-zinc-200 border border-zinc-700 rounded px-2 py-1 w-24 outline-none focus:border-primary"
        />
      </div>

      <div className="flex items-center gap-2">
        <span className="text-zinc-400">Service:</span>
        <input
          type="text"
          value={localFilter.ServiceID}
          onChange={(e) => updateField("ServiceID", e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="e.g. application"
          className="bg-zinc-800 text-zinc-200 border border-zinc-700 rounded px-2 py-1 w-28 outline-none focus:border-primary"
        />
      </div>

      <div className="flex items-center gap-2">
        <span className="text-zinc-400">Event:</span>
        <input
          type="text"
          value={localFilter.EventID}
          onChange={(e) => updateField("EventID", e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="e.g. CMD_START"
          className="bg-zinc-800 text-zinc-200 border border-zinc-700 rounded px-2 py-1 w-28 outline-none focus:border-primary"
        />
      </div>

      <div className="flex items-center gap-2 flex-1">
        <span className="text-zinc-400">Search:</span>
        <input
          type="text"
          value={localFilter.SearchText}
          onChange={(e) => updateField("SearchText", e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Search in message..."
          className="bg-zinc-800 text-zinc-200 border border-zinc-700 rounded px-2 py-1 w-full outline-none focus:border-primary"
        />
      </div>

      <button
        onClick={handleSearch}
        disabled={isLoading}
        className="bg-primary hover:bg-blue-600 text-white px-4 py-1 rounded flex items-center gap-2 transition-colors disabled:opacity-50"
      >
        <span className="material-symbols-rounded text-sm">search</span>
        {isLoading ? "Searching..." : "Search"}
      </button>
    </div>
  );
}

function LogConsole() {
  const [messages, selectedIdx] = useLogs(
    useShallow((s) => [s.messages, s.selectedIdx]),
  );

  if (selectedIdx === undefined) {
    return (
      <div className="flex-1 flex items-center justify-center text-zinc-600 font-mono">
        Select a log archive from the sidebar
      </div>
    );
  }

  if (messages.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center text-zinc-600 font-mono">
        No logs match the current filters
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-y-auto p-4 font-mono text-sm leading-relaxed">
      {messages.map((msg, i) => (
        <LogLine key={i} msg={msg} />
      ))}
    </div>
  );
}

function LogLine({ msg }: { msg: domain.LogMessage }) {
  // Parse date safely
  let dateStr = msg.Timestamp;
  try {
    const d = new Date(msg.Timestamp);
    // Format: HH:MM:SS.mmm
    dateStr = d.toISOString().split("T")[1].replace("Z", "");
  } catch (e) {
    // Ignore
  }

  const isError = msg.Level.toUpperCase() === "ERROR";
  const isWarn = msg.Level.toUpperCase() === "WARN";
  const isInfo = msg.Level.toUpperCase() === "INFO";

  return (
    <div className="flex gap-3 hover:bg-zinc-900/50 py-0.5 px-2 rounded group">
      <span className="text-zinc-500 shrink-0 select-none">[{dateStr}]</span>
      
      <span
        className={cn("w-12 shrink-0 font-bold", {
          "text-red-400": isError,
          "text-yellow-400": isWarn,
          "text-blue-400": isInfo,
          "text-zinc-400": !isError && !isWarn && !isInfo,
        })}
      >
        {msg.Level.toUpperCase().padEnd(5, " ")}
      </span>

      <div className="flex flex-col gap-0.5 w-full">
        <div className="flex items-center gap-2 text-xs opacity-70 group-hover:opacity-100 transition-opacity select-none">
          {msg.InstanceID && (
            <span className="bg-zinc-800 text-zinc-300 px-1.5 rounded">
              {msg.InstanceID}
            </span>
          )}
          {msg.ServiceID && (
            <span className="bg-zinc-800 text-zinc-300 px-1.5 rounded">
              {msg.ServiceID}
            </span>
          )}
          {msg.EventID && (
            <span className="bg-purple-900/50 text-purple-300 px-1.5 rounded border border-purple-800/50">
              {msg.EventID}
            </span>
          )}
        </div>
        <span
          className={cn("whitespace-pre-wrap break-words", {
            "text-red-300": isError,
            "text-zinc-200": !isError,
          })}
        >
          {msg.Message}
        </span>
      </div>
    </div>
  );
}
