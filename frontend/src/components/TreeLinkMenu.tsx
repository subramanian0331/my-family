import { displayName } from "./PersonCard";
import type { Person } from "../types";

export function TreeLinkMenu({
  from,
  to,
  x,
  y,
  busy,
  error,
  spouseDisabled,
  spouseDisabledReason,
  divorcedDisabled,
  divorcedDisabledReason,
  onSpouse,
  onDivorced,
  onChild,
  onCancel,
}: {
  from: Person;
  to: Person;
  x: number;
  y: number;
  busy: boolean;
  error: string | null;
  spouseDisabled?: boolean;
  spouseDisabledReason?: string | null;
  divorcedDisabled?: boolean;
  divorcedDisabledReason?: string | null;
  onSpouse: () => void;
  onDivorced: () => void;
  onChild: () => void;
  onCancel: () => void;
}) {
  return (
    <div
      className="pointer-events-auto absolute z-30 w-56 rounded-xl border border-[#e8e2d8] bg-white p-3 shadow-[0_8px_30px_rgba(60,50,40,0.18)]"
      style={{ left: x, top: y }}
      onPointerDown={(e) => e.stopPropagation()}
    >
      <p className="mb-2 text-xs text-[#8a8278]">
        Link <span className="font-medium text-[#2f2a26]">{displayName(from)}</span> to{" "}
        <span className="font-medium text-[#2f2a26]">{displayName(to)}</span>
      </p>
      {error && <p className="mb-2 text-xs text-red-600">{error}</p>}
      <div className="flex flex-col gap-1.5">
        <button
          type="button"
          disabled={busy || spouseDisabled}
          onClick={onSpouse}
          title={spouseDisabledReason ?? undefined}
          className="rounded-lg bg-[#fdf2f5] px-3 py-2 text-left text-sm font-medium text-[#b44d6a] hover:bg-[#fce8ee] disabled:cursor-not-allowed disabled:opacity-50"
        >
          Spouse
          {spouseDisabled && spouseDisabledReason && (
            <span className="mt-0.5 block text-xs font-normal text-[#8a8278]">
              {spouseDisabledReason}
            </span>
          )}
        </button>
        <button
          type="button"
          disabled={busy || divorcedDisabled}
          onClick={onDivorced}
          title={divorcedDisabledReason ?? undefined}
          className="rounded-lg bg-[#f5f3f8] px-3 py-2 text-left text-sm font-medium text-[#6b5b8a] hover:bg-[#ebe6f2] disabled:cursor-not-allowed disabled:opacity-50"
        >
          Former spouse (divorced)
          {divorcedDisabled && divorcedDisabledReason && (
            <span className="mt-0.5 block text-xs font-normal text-[#8a8278]">
              {divorcedDisabledReason}
            </span>
          )}
        </button>
        <button
          type="button"
          disabled={busy}
          onClick={onChild}
          className="rounded-lg bg-[#f3f8f5] px-3 py-2 text-left text-sm font-medium text-[#4a7c62] hover:bg-[#e8f2ec] disabled:opacity-50"
        >
          Child — {displayName(to)} is child of {displayName(from)}
        </button>
        <button
          type="button"
          disabled={busy}
          onClick={onCancel}
          className="rounded-lg px-3 py-1.5 text-sm text-[#8a8278] hover:bg-[#f3efe8]"
        >
          Cancel
        </button>
      </div>
    </div>
  );
}