import { zodResolver } from "@hookform/resolvers/zod";
import { LoaderCircle, Play, RadioTower } from "lucide-react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import type { Capabilities, CheckType, RunOptions } from "../api/types";

function hasNoControlCharacters(value: string) {
  return [...value].every((character) => {
    const code = character.charCodeAt(0);
    return code > 31 && code !== 127;
  });
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
});

type FormValues = z.infer<typeof schema>;

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
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      target: initialTarget,
      timeoutMs: capabilities?.runtime.defaultTimeoutMs ?? 5000,
    },
  });

  return (
    <form
      className="diagnostic-form"
      onSubmit={handleSubmit((values) =>
        onSubmit({
          target: values.target.trim(),
          checks: ["dns"],
          options: {
            timeoutMs: values.timeoutMs,
            followRedirects: true,
            maxRedirects: 5,
            ipVersion: "auto",
          },
        }),
      )}
    >
      <div className="target-row">
        <label className="target-field">
          <span>Target</span>
          <input
            {...register("target")}
            placeholder="example.com"
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
        <div>
          <p className="option-label">Checks</p>
          <label className="check-option enabled">
            <input type="checkbox" checked readOnly />
            <RadioTower size={17} />
            DNS
            <span>available</span>
          </label>
          {["TCP", "HTTP", "TLS"].map((check) => (
            <label className="check-option" key={check} title="Coming next">
              <input type="checkbox" disabled />
              {check}
              <span>soon</span>
            </label>
          ))}
        </div>
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
