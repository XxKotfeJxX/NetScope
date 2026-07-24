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
  httpMethod: z.enum(["GET", "HEAD"]),
  followRedirects: z.boolean(),
  maxRedirects: z.number().int().min(0).max(10),
  ipVersion: z.enum(["auto", "ipv4", "ipv6"]),
  pingPackets: z.number().int().min(1).max(10),
  maxHops: z.number().int().min(1).max(64),
});

type FormValues = z.infer<typeof schema>;

const checkDefinitions: Array<{ type: CheckType; label: string }> = [
  { type: "dns", label: "DNS" },
  { type: "tcp", label: "TCP" },
  { type: "tls", label: "TLS" },
  { type: "http", label: "HTTP" },
  { type: "ping", label: "Ping" },
  { type: "traceroute", label: "Trace" },
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
      httpMethod: "GET",
      followRedirects: true,
      maxRedirects: 5,
      ipVersion: "auto",
      pingPackets: 4,
      maxHops: 20,
    },
  });
  const timeoutMs = useWatch({ control, name: "timeoutMs" });
  const ipVersion = useWatch({ control, name: "ipVersion" });
  const followRedirects = useWatch({ control, name: "followRedirects" });

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
            httpMethod: values.httpMethod,
            followRedirects: values.followRedirects,
            maxRedirects: values.maxRedirects,
            ipVersion: values.ipVersion,
            pingPackets: values.pingPackets,
            maxHops: values.maxHops,
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
          const available = capabilities
            ? (capabilities.checks[type]?.available ?? false)
            : type !== "ping" && type !== "traceroute";
          const selected = checks.includes(type);
          return (
            <label
              className={`check-option ${selected ? "enabled" : ""} ${
                available ? "" : "unavailable"
              }`}
              key={type}
            >
              <input
                type="checkbox"
                checked={selected}
                disabled={!available}
                onChange={() => toggleCheck(type)}
                title={
                  available
                    ? undefined
                    : capabilities?.checks[type]?.reason || "Unavailable"
                }
              />
              <span className="check-symbol" aria-hidden="true">
                {selected ? "✓" : "○"}
              </span>
              {label}
            </label>
          );
        })}
      </div>
      {checkError && <p className="field-error">{checkError}</p>}

      <div className="option-summary">
        <span>Timeout {timeoutMs / 1000}s</span>
        <span>
          {ipVersion === "auto" ? "IPv4 / IPv6 auto" : ipVersion.toUpperCase()}
        </span>
        <span>{followRedirects ? "Follow redirects" : "Stop on redirect"}</span>
      </div>

      <details className="advanced-options">
        <summary>Advanced options</summary>
        <div className="advanced-sections">
          <fieldset className="option-group">
            <legend>Connection</legend>
            <div className="advanced-grid">
              <label>
                <span className="option-label">IP preference</span>
                <select {...register("ipVersion")}>
                  <option value="auto">Automatic</option>
                  <option value="ipv4">IPv4 only</option>
                  <option value="ipv6">IPv6 only</option>
                </select>
              </label>
              <label>
                <span className="option-label">Probe timeout</span>
                <select {...register("timeoutMs", { valueAsNumber: true })}>
                  <option value={2000}>2 seconds</option>
                  <option value={5000}>5 seconds</option>
                  <option value={10000}>10 seconds</option>
                  <option value={30000}>30 seconds</option>
                </select>
              </label>
              <label className="advanced-span">
                <span className="option-label">TCP ports</span>
                <input {...register("tcpPorts")} placeholder="80, 443" />
                {errors.tcpPorts && (
                  <span className="field-error">{errors.tcpPorts.message}</span>
                )}
              </label>
            </div>
          </fieldset>

          <fieldset className="option-group">
            <legend>HTTP</legend>
            <div className="advanced-grid">
              <label>
                <span className="option-label">Method</span>
                <select {...register("httpMethod")}>
                  <option value="GET">GET</option>
                  <option value="HEAD">HEAD</option>
                </select>
              </label>
              <label>
                <span className="option-label">Maximum redirects</span>
                <input
                  type="number"
                  {...register("maxRedirects", { valueAsNumber: true })}
                />
              </label>
              <label className="toggle-field advanced-span">
                <input type="checkbox" {...register("followRedirects")} />
                <span>Follow HTTP redirects</span>
              </label>
            </div>
          </fieldset>

          <fieldset className="option-group">
            <legend>Route probes</legend>
            <div className="advanced-grid">
              <label>
                <span className="option-label">Ping packets</span>
                <input
                  type="number"
                  {...register("pingPackets", { valueAsNumber: true })}
                />
              </label>
              <label>
                <span className="option-label">Traceroute hops</span>
                <input
                  type="number"
                  {...register("maxHops", { valueAsNumber: true })}
                />
              </label>
            </div>
          </fieldset>
        </div>
      </details>
    </form>
  );
}
