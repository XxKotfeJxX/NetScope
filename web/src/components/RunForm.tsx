import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import type { Capabilities, CheckType, RunOptions } from "../api/types";

function hasNoControlCharacters(value: string) {
  return [...value].every((character) => {
    const code = character.charCodeAt(0);
    return code > 31 && code !== 127;
  });
}

function validPorts(value: string) {
  if (value.trim() === "") return true;
  const ports = value.split(",").map((item) => Number(item.trim()));
  return (
    ports.length <= 10 &&
    ports.every((port) => Number.isInteger(port) && port >= 1 && port <= 65535)
  );
}

const schema = z.object({
  target: z
    .string()
    .trim()
    .min(1, "Enter a hostname, URL, or IP address.")
    .max(2048, "Target must be 2,048 characters or fewer.")
    .refine(hasNoControlCharacters, {
      message: "Control characters are not allowed.",
    })
    .refine((value) => !/^[^/\s]+\/\d{1,3}$/.test(value), {
      message: "CIDR ranges are not supported.",
    }),
  timeoutMs: z.number().min(500).max(30_000),
  tcpPorts: z
    .string()
    .refine(validPorts, "Use up to 10 ports between 1 and 65,535."),
});

type FormValues = z.infer<typeof schema>;

const checkDefinitions: Array<{ type: CheckType; label: string }> = [
  { type: "dns", label: "DNS" },
  { type: "tcp", label: "TCP" },
  { type: "tls", label: "TLS" },
  { type: "http", label: "HTTP" },
];

export function RunForm({
  capabilities,
  initialTarget = "",
  pending,
  onSubmit,
}: {
  capabilities?: Capabilities;
  initialTarget?: string;
  pending: boolean;
  onSubmit: (payload: {
    target: string;
    checks: CheckType[];
    options: RunOptions;
  }) => void;
}) {
  const [checks, setChecks] = useState<CheckType[]>([
    "dns",
    "tcp",
    "http",
    "tls",
  ]);
  const [checkError, setCheckError] = useState("");
  const {
    register,
    handleSubmit,
    control,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      target: initialTarget,
      timeoutMs: capabilities?.runtime.defaultTimeoutMs ?? 5000,
      tcpPorts: "80, 443",
    },
  });
  const timeoutMs = useWatch({ control, name: "timeoutMs" });

  function toggleCheck(type: CheckType) {
    setChecks((current) =>
      current.includes(type)
        ? current.filter((check) => check !== type)
        : [...current, type],
    );
    setCheckError("");
  }

  return (
    <form
      className="diagnostic-form"
      onSubmit={handleSubmit((values) => {
        if (checks.length === 0) {
          setCheckError("Select at least one diagnostic check.");
          return;
        }
        onSubmit({
          target: values.target.trim(),
          checks,
          options: {
            timeoutMs: values.timeoutMs,
            tcpPorts: values.tcpPorts
              .split(",")
              .map((port) => Number(port.trim()))
              .filter(Boolean),
            httpMethod: "GET",
            followRedirects: true,
            maxRedirects: 5,
            ipVersion: "auto",
          },
        });
      })}
    >
      <div className="target-row">
        <label className="target-field">
          <span className="sr-only">Target</span>
          <input
            {...register("target")}
            placeholder="example.com, https://api.example.com, or 1.1.1.1"
            autoComplete="off"
            aria-invalid={Boolean(errors.target)}
          />
        </label>
        <button className="primary-button" type="submit" disabled={pending}>
          {pending && <span className="button-spinner" aria-hidden="true" />}
          {pending ? "Starting…" : "Run diagnostic"}
        </button>
      </div>
      {errors.target && <p className="field-error">{errors.target.message}</p>}

      <div className="check-strip" aria-label="Diagnostic checks">
        {checkDefinitions.map(({ type, label }) => {
          const available = capabilities?.checks[type]?.available ?? true;
          const selected = checks.includes(type);
          return (
            <label
              className={`check-option ${selected ? "enabled" : ""}`}
              key={type}
            >
              <input
                type="checkbox"
                checked={selected}
                disabled={!available}
                onChange={() => toggleCheck(type)}
              />
              <span className="check-symbol" aria-hidden="true">
                {selected ? "✓" : "○"}
              </span>
              {label}
            </label>
          );
        })}
        <span className="check-option unavailable" aria-disabled="true">
          <span className="check-symbol">○</span> Ping <small>v0.2</small>
        </span>
        <span className="check-option unavailable" aria-disabled="true">
          <span className="check-symbol">○</span> Trace <small>v0.2</small>
        </span>
      </div>
      {checkError && <p className="field-error">{checkError}</p>}

      <div className="option-summary">
        <span>Timeout {timeoutMs / 1000}s</span>
        <span>IPv4 / IPv6 auto</span>
        <span>Follow redirects</span>
      </div>

      <details className="advanced-options">
        <summary>Advanced options</summary>
        <div className="advanced-grid">
          <label className="ports-field">
            <span className="option-label">TCP ports</span>
            <input {...register("tcpPorts")} placeholder="80, 443" />
            {errors.tcpPorts && (
              <span className="field-error">{errors.tcpPorts.message}</span>
            )}
          </label>
          <label className="timeout-field">
            <span className="option-label">Probe timeout</span>
            <select {...register("timeoutMs", { valueAsNumber: true })}>
              <option value={2000}>2 seconds</option>
              <option value={5000}>5 seconds</option>
              <option value={10000}>10 seconds</option>
              <option value={30000}>30 seconds</option>
            </select>
          </label>
        </div>
      </details>
    </form>
  );
}
