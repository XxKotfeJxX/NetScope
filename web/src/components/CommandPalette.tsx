import { Search } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";

type Command = {
  id: string;
  label: string;
  hint: string;
  keywords: string;
  run: () => void;
};

export function CommandPalette({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const navigate = useNavigate();
  const { account, selectWorkspace } = useAuth();
  const inputRef = useRef<HTMLInputElement>(null);
  const returnFocusRef = useRef<HTMLElement | null>(null);
  const [query, setQuery] = useState("");
  const [activeIndex, setActiveIndex] = useState(0);
  const close = useCallback(() => {
    setQuery("");
    setActiveIndex(0);
    onOpenChange(false);
  }, [onOpenChange]);

  const commands = useMemo<Command[]>(() => {
    const closeAndNavigate = (path: string) => {
      close();
      navigate(path);
    };
    return [
      {
        id: "diagnose",
        label: "Run diagnostic",
        hint: "Diagnose",
        keywords: "inspect probe check target",
        run: () => closeAndNavigate("/"),
      },
      {
        id: "targets",
        label: "Open target",
        hint: "Targets",
        keywords: "saved target monitoring",
        run: () => closeAndNavigate("/targets"),
      },
      {
        id: "history",
        label: "Search history",
        hint: "History",
        keywords: "runs reports search",
        run: () => closeAndNavigate("/history"),
      },
      {
        id: "schedule",
        label: "Create schedule",
        hint: "Monitoring",
        keywords: "new target interval scheduled",
        run: () => closeAndNavigate("/targets?new=1"),
      },
      ...(account?.workspaces.map((workspace) => ({
        id: `workspace-${workspace.id}`,
        label: `Switch workspace · ${workspace.name}`,
        hint:
          workspace.id === account.activeWorkspace.id
            ? "Current"
            : workspace.role,
        keywords: `workspace ${workspace.name} ${workspace.slug}`,
        run: () => {
          close();
          void selectWorkspace(workspace.id);
        },
      })) ?? []),
    ];
  }, [account, close, navigate, selectWorkspace]);

  const filtered = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return commands;
    return commands.filter((command) =>
      `${command.label} ${command.keywords}`.toLowerCase().includes(normalized),
    );
  }, [commands, query]);
  const selectedIndex =
    filtered.length === 0 ? 0 : Math.min(activeIndex, filtered.length - 1);

  useEffect(() => {
    function handleShortcut(event: KeyboardEvent) {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        if (open) close();
        else onOpenChange(true);
      }
    }
    window.addEventListener("keydown", handleShortcut);
    return () => window.removeEventListener("keydown", handleShortcut);
  }, [close, onOpenChange, open]);

  useEffect(() => {
    if (!open) return;
    returnFocusRef.current = document.activeElement as HTMLElement;
    window.requestAnimationFrame(() => inputRef.current?.focus());
    return () => returnFocusRef.current?.focus();
  }, [open]);

  if (!open) return null;

  return (
    <div
      className="command-backdrop"
      role="presentation"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) close();
      }}
    >
      <section
        className="command-palette"
        role="dialog"
        aria-modal="true"
        aria-label="Command palette"
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            event.preventDefault();
            close();
          } else if (event.key === "ArrowDown") {
            event.preventDefault();
            setActiveIndex((current) =>
              filtered.length ? (current + 1) % filtered.length : 0,
            );
          } else if (event.key === "ArrowUp") {
            event.preventDefault();
            setActiveIndex((current) =>
              filtered.length
                ? (current - 1 + filtered.length) % filtered.length
                : 0,
            );
          } else if (event.key === "Enter" && filtered[selectedIndex]) {
            event.preventDefault();
            filtered[selectedIndex].run();
          } else if (event.key === "Tab") {
            event.preventDefault();
            inputRef.current?.focus();
          }
        }}
      >
        <label className="command-search">
          <Search aria-hidden="true" size={17} strokeWidth={1.7} />
          <span className="sr-only">Search commands</span>
          <input
            ref={inputRef}
            value={query}
            placeholder="Type a command or target…"
            autoComplete="off"
            role="combobox"
            aria-expanded="true"
            aria-controls="command-results"
            aria-activedescendant={
              filtered[selectedIndex]
                ? `command-${filtered[selectedIndex].id}`
                : undefined
            }
            onChange={(event) => {
              setQuery(event.target.value);
              setActiveIndex(0);
            }}
          />
          <kbd>ESC</kbd>
        </label>
        <div
          className="command-results"
          id="command-results"
          role="listbox"
          aria-label="Available commands"
        >
          {filtered.map((command, index) => (
            <button
              className={index === selectedIndex ? "is-active" : ""}
              id={`command-${command.id}`}
              key={command.id}
              type="button"
              role="option"
              aria-selected={index === selectedIndex}
              onMouseEnter={() => setActiveIndex(index)}
              onClick={command.run}
            >
              <span>{command.label}</span>
              <small>{command.hint}</small>
            </button>
          ))}
          {filtered.length === 0 && (
            <p className="command-empty">No matching command.</p>
          )}
        </div>
        <footer>
          <span>
            <kbd>↑</kbd>
            <kbd>↓</kbd> navigate
          </span>
          <span>
            <kbd>↵</kbd> open
          </span>
        </footer>
      </section>
    </div>
  );
}
