/* eslint-disable @typescript-eslint/no-explicit-any */
import { useShallow } from "zustand/shallow";
import Button from "../components/button";
import { useLogs } from "../hooks/useLogs";
import { cn } from "../utils/cn";
import { useState } from "react";

export default function LogsPage() {
  const [loadAll, logPaths, file, loadFile] = useLogs(
    useShallow((s) => [s.loadAll, s.logPaths, s.file, s.loadFile]),
  );

  return (
    <div className="h-[calc(100vh-60px)] flex overflow-clip">
      <aside className="bg-cwhite border-r border-border h-full flex flex-col overflow-hidden w-70 select-none">
        <header className="bg-gray bordre-b border-border px-3 py-1.5 flex items-center justify-between">
          <h4>Logs</h4>
          <Button icon="refresh" onClick={loadAll} />
        </header>
        <main className="p-3 flex-1 flex flex-col gap-2 overflow-y-scroll overflow-x-auto">
          {logPaths.map((e, i) => (
            <LogPathCard
              i={i}
              name={e[0]}
              onClick={(path) => loadFile(`${e[1]}/${path}`)}
            />
          ))}
        </main>
      </aside>
      <main className="flex-1 overflow-y-auto">
        {file && (
          <div className="rounded-md border-border bg-cwhite p-3 whitespace-pre-wrap">
            {file}
          </div>
        )}
      </main>
    </div>
  );
}

function LogPathCard({
  i,
  name,
  onClick,
}: {
  i: number;
  name: string;
  onClick(value: string): void;
}) {
  const selectedIdx = useLogs((s) => s.selectedIdx);
  const selected = i === (selectedIdx ?? -1);
  const current = useLogs((s) => s.current);

  return (
    <div>
      <div
        onClick={() => useLogs.getState().loadLogs(selected ? undefined : i)}
        className={cn(
          `hoverable-gray text-cblack flex items-center gap-2 px-3 py-1.5 cursor-pointer rounded-md`,
          {
            "hoverable-primary text-onPrimary bg-primary": selected,
          },
        )}
      >
        <span className="material-symbols-rounded">folder</span>
        {name}
      </div>
      {selected && current && <FileExplorer data={current} onClick={onClick} />}
    </div>
  );
}

function FileExplorer({
  data,
  onClick,
}: {
  data: Record<string, any>;
  onClick(value: string): void;
}) {
  return (
    <div className="px-3 py-1.5">
      {Object.entries(data).map(([key, value]) => (
        <TreeNode key={key} name={key} data={value} onClick={onClick} />
      ))}
    </div>
  );
}

function TreeNode({
  name,
  data,
  onClick,
}: {
  name: string;
  data: any;
  onClick(value: string): void;
}) {
  const [isOpen, setIsOpen] = useState(true);
  const isFolder =
    data !== null && typeof data === "object" && !Array.isArray(data);
  const isFileList = Array.isArray(data);

  return (
    <div className="select-none">
      {isFolder && (
        <div
          onClick={() => setIsOpen((s) => !s)}
          className="flex items-center gap-2 px-3 py-1.5 pl-2 hoverable-gray rounded cursor-pointer transition-colors"
        >
          <span className="text-gray-400 material-symbols-rounded">
            {isOpen ? "keyboard_arrow_down" : "chevron_right"}
          </span>
          <span className="material-symbols-rounded text-amber-500 fill-amber-500/20">
            {isOpen ? "folder_open" : "folder"}
          </span>
          <span className="text-sm font-medium text-gray-700">{name}</span>
        </div>
      )}

      {isOpen && (
        <>
          {isFolder && (
            <div className="ml-4.5 border-l border-border pl-2">
              {Object.entries(data).map(([key, value]) => (
                <TreeNode
                  key={key}
                  name={key}
                  data={value}
                  onClick={(e) => onClick(`${name}/${e}`)}
                />
              ))}
            </div>
          )}

          {isFileList &&
            data.map((fileName, index) => (
              <div
                onClick={() => onClick(fileName)}
                key={index}
                className="flex items-center gap-2 px-3 py-1.5 text-dark-gray hoverable-gray rounded cursor-pointer w-max"
              >
                <span className="text-blue-400 material-symbols-rounded">
                  description
                </span>
                <span className="text-sm font-mono">{fileName}</span>
              </div>
            ))}
        </>
      )}
    </div>
  );
}
