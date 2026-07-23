import { zodResolver } from "@hookform/resolvers/zod";
import {
  Braces,
  Globe2,
  LoaderCircle,
  LockKeyhole,
  Network,
  Play,
} from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
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

const checkDefinitions: Array<{
  type: CheckType;
  label: string;
  icon: typeof Braces;
}> = [
  { type: "dns", label: "DNS", icon: Braces },
  { type: "tcp", label: "TCP", icon: Network },
  { type: "http", label: "HTTP", icon: Globe2 },
  { type: "tls", label: "TLS", icon: LockKeyhole },
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
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      target: initialTarget,
      timeoutMs: capabilities?.runtime.defaultTimeoutMs ?? 5000,
      tcpPorts: "80, 443",
    },
  });

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
          <span>Target</span>
          <input
            {...register("target")}
            placeholder="https://example.com"
            autoComplete="off"
            aria-invalid={Boolean(errors.target)}
          />
        </label>
        <button className="primary-button" type="submit" disabled={pending}>
          {pending ? (
            <LoaderCircle className="spin" size={18} />
          ) : (
            <Play size={18} fill="currentColor" />
          )}
          {pending ? "Starting…" : "Run diagnostics"}
        </button>
      </div>
      {errors.target && <p className="field-error">{errors.target.message}</p>}

      <div className="form-options">
        <div className="checks-group">
          <p className="option-label">Checks</p>
          {checkDefinitions.map(({ type, label, icon: Icon }) => {
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
                <Icon size={16} />
                {label}
                <span>{available ? (selected ? "on" : "off") : "n/a"}</span>
              </label>
            );
          })}
          {checkError && <p className="field-error">{checkError}</p>}
        </div>
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
    </form>
  );
}
