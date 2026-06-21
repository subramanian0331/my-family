import type { Person } from "../types";
import { isDeceased, lifeSpan } from "../lib/personUtils";
import { displayName } from "./PersonCard";
import { PersonPhoto } from "./PersonPhoto";

type AvatarStyle = {
  gradient: string;
  icon: string;
  ring: string;
};

function avatarStyle(person: Person): AvatarStyle {
  if (person.married_in) {
    return {
      gradient: "from-[#c9b8e8] via-[#b8a2db] to-[#9b7fc9]",
      icon: "text-white/95",
      ring: "ring-[#e2d4f4]",
    };
  }
  if (person.gender === "female") {
    return {
      gradient: "from-[#f5b8c8] via-[#ec9bb0] to-[#d97a94]",
      icon: "text-white/95",
      ring: "ring-[#f8d4de]",
    };
  }
  if (person.gender === "male") {
    return {
      gradient: "from-[#8eb8d8] via-[#6fa3c8] to-[#4f86ad]",
      icon: "text-white/95",
      ring: "ring-[#cfe3f2]",
    };
  }
  return {
    gradient: "from-[#c8c2ba] via-[#b0aaa2] to-[#969088]",
    icon: "text-white/90",
    ring: "ring-[#e4dfd8]",
  };
}

function DefaultAvatar({ person }: { person: Person }) {
  const style = avatarStyle(person);
  const isFemale = person.gender === "female";

  return (
    <div
      className={`flex h-full w-full items-center justify-center bg-gradient-to-br ${style.gradient} ${style.ring} ring-2`}
    >
      <svg viewBox="0 0 64 64" className={`h-[58%] w-[58%] ${style.icon}`} aria-hidden>
        <circle cx="32" cy="24" r="11" fill="currentColor" opacity="0.95" />
        {isFemale ? (
          <path
            d="M14 58c2.5-12 9-18 18-18s15.5 6 18 18"
            fill="currentColor"
            opacity="0.9"
          />
        ) : (
          <path
            d="M12 58c3-11 10-17 20-17s17 6 20 17"
            fill="currentColor"
            opacity="0.9"
          />
        )}
      </svg>
    </div>
  );
}

export function TreePersonNode({
  person,
  onClick,
  onContextMenu,
  canEdit,
  highlight,
  onLinkDragStart,
}: {
  person: Person;
  onClick: () => void;
  onContextMenu?: (e: React.MouseEvent) => void;
  canEdit?: boolean;
  highlight?: "source" | "target" | null;
  onLinkDragStart?: (e: React.PointerEvent) => void;
}) {
  const dates = lifeSpan(person);
  const hasNotes = Boolean(person.notes?.trim());
  const marriedIn = Boolean(person.married_in);
  const deceased = isDeceased(person);
  const avatar = avatarStyle(person);

  return (
    <button
      type="button"
      data-tree-node
      data-person-id={person.id}
      onClick={onClick}
      onContextMenu={onContextMenu}
      title={marriedIn ? "Married into this family — right-click to open their family tree" : undefined}
      className={`group relative flex h-full w-full flex-col items-center rounded-2xl px-2 pb-2.5 pt-2.5 text-center transition hover:-translate-y-0.5 ${
        marriedIn
          ? "bg-[#faf6fd] shadow-[0_4px_16px_rgba(120,70,150,0.12)] ring-2 ring-[#d4bde8] hover:shadow-[0_8px_24px_rgba(120,70,150,0.18)]"
          : deceased
            ? "bg-[#f4f4f5] shadow-[0_4px_16px_rgba(60,60,70,0.08)] ring-1 ring-[#d4d4d8] hover:shadow-[0_8px_24px_rgba(60,60,70,0.1)]"
            : "bg-white shadow-[0_4px_16px_rgba(30,45,60,0.08)] ring-1 ring-[#d8e0e8] hover:shadow-[0_8px_24px_rgba(30,45,60,0.12)]"
      } ${
        highlight === "source"
          ? "!ring-2 !ring-[#d45d7a] shadow-[0_0_0_4px_rgba(212,93,122,0.2)]"
          : highlight === "target"
            ? "!ring-2 !ring-[#4a9a6a] shadow-[0_0_0_4px_rgba(74,154,106,0.2)]"
            : ""
      }`}
    >
      {marriedIn && (
        <span className="absolute -top-2 left-1/2 z-10 -translate-x-1/2 whitespace-nowrap rounded-full bg-[#8f6bab] px-2 py-0.5 text-[9px] font-semibold uppercase tracking-wide text-white shadow-sm">
          Married in
        </span>
      )}
      {deceased && !marriedIn && (
        <span className="absolute -top-2 left-1/2 z-10 -translate-x-1/2 whitespace-nowrap rounded-full bg-[#71717a] px-2 py-0.5 text-[9px] font-semibold uppercase tracking-wide text-white shadow-sm">
          Deceased
        </span>
      )}
      <div className={`relative mb-2 h-[74px] w-[74px] ${deceased ? "opacity-80 grayscale" : ""}`}>
        {person.photo_id ? (
          <PersonPhoto
            photoId={person.photo_id}
            alt=""
            className={`h-full w-full rounded-full object-cover ring-2 ${avatar.ring}`}
            fallback={
              <div className="h-full w-full overflow-hidden rounded-full">
                <DefaultAvatar person={person} />
              </div>
            }
          />
        ) : (
          <div className="h-full w-full overflow-hidden rounded-full">
            <DefaultAvatar person={person} />
          </div>
        )}
        {hasNotes && (
          <span
            className="absolute -bottom-0.5 -right-0.5 flex h-5 w-5 items-center justify-center rounded-full bg-[#4a9a6a] text-white shadow-sm"
            title="Has notes"
          >
            <svg viewBox="0 0 16 16" className="h-3 w-3" fill="currentColor" aria-hidden>
              <path d="M3 2.5A1.5 1.5 0 0 1 4.5 1h5A1.5 1.5 0 0 1 11 2.5v7.2l-2.7 2.2a.8.8 0 0 1-1.3-.6V2.5z" />
            </svg>
          </span>
        )}
      </div>
      <div className="w-full px-1">
        <div
          className={`truncate text-[13px] font-semibold leading-tight ${
            marriedIn ? "text-[#5c3d6e]" : deceased ? "text-[#52525b]" : "text-[#1e2a36]"
          }`}
        >
          {displayName(person)}
        </div>
        {dates && (
          <div
            className={`mt-0.5 text-[11px] ${
              marriedIn ? "text-[#8f6bab]" : deceased ? "text-[#71717a]" : "text-[#5c6b78]"
            }`}
          >
            {dates}
          </div>
        )}
      </div>
      {canEdit && onLinkDragStart && (
        <span
          role="button"
          tabIndex={-1}
          data-link-handle
          title="Drag to link spouse or child"
          onPointerDown={(e) => {
            e.stopPropagation();
            e.preventDefault();
            onLinkDragStart(e);
          }}
          className="absolute -bottom-1 -right-1 z-20 flex h-6 w-6 cursor-crosshair items-center justify-center rounded-full border border-[#c5d0da] bg-white text-[#5c6b78] opacity-0 shadow-md transition group-hover:opacity-100 hover:border-[#d45d7a] hover:text-[#d45d7a]"
        >
          <svg viewBox="0 0 16 16" className="h-3.5 w-3.5" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden>
            <path d="M8 3v10M3 8h10" strokeLinecap="round" />
          </svg>
        </span>
      )}
    </button>
  );
}