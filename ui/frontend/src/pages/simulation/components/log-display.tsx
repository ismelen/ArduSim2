/* eslint-disable @typescript-eslint/no-explicit-any */
import { useEffect, useRef, useState } from "react";
import {
  EventsOff,
  EventsOn,
  EventsOnce,
} from "../../../../wailsjs/runtime/runtime";
import Button from "../../../components/button";
import { cn } from "../../../utils/cn";

declare global {
  interface Window {
    wails?: any;
    runtime?: any;
  }
}

interface LogEntry {
  time: string;
  level: string;
  msg: string;
  levelClass: string;
}

const EVENT_TAGS = [
  {
    tag: "simulation:log",
    level: "[MSG]",
    levelClass: "text-primary",
    getMsg: (msg: any) => msg as string,
  },
  {
    tag: "netsim:message",
    level: "[MSG]",
    levelClass: "text-amber-600",
    getMsg: (msg: any) => msg.label as string,
  },
];

interface Props {
  className?: string;
  onFinishReceived(): void;
}

export default function LogDisplay({ className, onFinishReceived }: Props) {
  const [isOpen, setIsOpen] = useState(true);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const logEndRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!window.wails && !window.runtime) {
      console.warn(
        "Wails runtime no detectado. Simulando entorno de navegador.",
      );
      return;
    }
    
    for (const eventType of EVENT_TAGS) {
      EventsOn(eventType.tag, (msg: any) => {
        const now = new Date();
        const timeStr = now.toLocaleTimeString([], { hour12: false });

        setLogs((s) => [
          ...s.slice(-49),
          {
            time: timeStr,
            level: eventType.level,
            msg: eventType.getMsg(msg),
            levelClass: eventType.levelClass,
          },
        ]);
      });
    }

    EventsOnce("simulation:finished", onFinishReceived);

    return () => {
      for (const eventType of EVENT_TAGS) {
        EventsOff(eventType.tag);
      }
    };
  }, [onFinishReceived]);

  useEffect(() => {
    if (!isOpen) return;
    logEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logs, isOpen]);

  return (
    <div
      className={cn(
        `h-1/3 bg-cwhite
          z-50 rounded-md border-border border overflow-y-auto`,
        className,
        {
          "h-min": !isOpen,
        },
      )}
    >
      <span
        onClick={() => setIsOpen((s) => !s)}
        className={cn(
          `bg-gray border-b border-border flex items-center 
          justify-between p-1 pl-3 cursor-pointer sticky top-0`,
          { "border-transparent": !isOpen },
        )}
      >
        <p className="text-base font-medium">Log</p>
        <Button icon={isOpen ? "keyboard_arrow_down" : "keyboard_arrow_up"} />
      </span>
      {isOpen && (
        <div className="h-full overlow-y-auto">
          {logs.map((log, i) => {
            return (
              <div key={i} className="flex flex-row gap-2 text-dark-gray px-2">
                <span className="text-cblack">{log.time}</span>
                <span className={cn("text-lg", log.levelClass)}>
                  {log.level}
                </span>
                {log.msg}
              </div>
            );
          })}
          <div ref={logEndRef} />
        </div>
      )}
    </div>
  );
}
