import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../api/client";
import type { Person, PersonFamilyRef, Relationship } from "../types";
import { alreadyLinkedAsSpouses, isAvailableSpousePartner } from "../lib/spouseFilter";
import { computeTreeLayout, NODE_HEIGHT, NODE_WIDTH } from "../lib/treeLayout";
import { TreeAddPersonPanel } from "./TreeAddPersonPanel";
import { TreeLinkMenu } from "./TreeLinkMenu";
import { TreePersonNode } from "./TreePersonNode";

interface TreeViewProps {
  persons: Person[];
  relationships: Relationship[];
  familyId: string;
  canEdit?: boolean;
  focusId?: string;
  className?: string;
  onSelect: (person: Person) => void;
  onRelationshipsChanged: () => void | Promise<void>;
}

type Transform = { x: number; y: number; scale: number };

type LinkDrag = {
  fromId: string;
  x1: number;
  y1: number;
  x2: number;
  y2: number;
};

type LinkMenuState = {
  fromId: string;
  toId: string;
  x: number;
  y: number;
};

type MarriedInMenuState = {
  person: Person;
  families: PersonFamilyRef[];
  x: number;
  y: number;
};

const MIN_SCALE = 0.2;
const MAX_SCALE = 2.5;
const CANVAS_PADDING = 40;

function formatLinkError(err: unknown) {
  const msg = err instanceof Error ? err.message : "Failed to create link";
  if (msg.includes("already has a spouse") || msg.includes("already married to")) {
    return `${msg}. Unlink the current spouse first.`;
  }
  return msg;
}

export function TreeView({
  persons,
  relationships,
  familyId,
  canEdit = false,
  className = "",
  onSelect,
  onRelationshipsChanged,
}: TreeViewProps) {
  const navigate = useNavigate();
  const containerRef = useRef<HTMLDivElement>(null);
  const layout = useMemo(() => computeTreeLayout(persons, relationships), [persons, relationships]);
  const personById = useMemo(() => new Map(persons.map((p) => [p.id, p])), [persons]);

  const [transform, setTransform] = useState<Transform>({ x: 0, y: 0, scale: 1 });
  const [animating, setAnimating] = useState(false);
  const dragging = useRef(false);
  const lastPointer = useRef({ x: 0, y: 0 });
  const velocity = useRef({ x: 0, y: 0 });
  const inertiaFrame = useRef<number | null>(null);

  const [linkDrag, setLinkDrag] = useState<LinkDrag | null>(null);
  const [linkHoverId, setLinkHoverId] = useState<string | null>(null);
  const [linkMenu, setLinkMenu] = useState<LinkMenuState | null>(null);
  const [marriedInMenu, setMarriedInMenu] = useState<MarriedInMenuState | null>(null);
  const [marriedInLoading, setMarriedInLoading] = useState(false);
  const [linkBusy, setLinkBusy] = useState(false);
  const [linkError, setLinkError] = useState<string | null>(null);
  const [addPersonOpen, setAddPersonOpen] = useState(false);
  const [addPersonAnchorId, setAddPersonAnchorId] = useState<string | null>(null);
  const suppressNodeClickRef = useRef(false);

  const withAnimation = useCallback((fn: () => void) => {
    setAnimating(true);
    fn();
    window.setTimeout(() => setAnimating(false), 280);
  }, []);

  const fitToView = useCallback(
    (smooth = false) => {
      const el = containerRef.current;
      if (!el || layout.width === 0) return;
      const rect = el.getBoundingClientRect();
      const padding = 40;
      const scale = Math.min(
        (rect.width - padding) / layout.width,
        (rect.height - padding) / layout.height,
      );
      const clamped = Math.max(MIN_SCALE, Math.min(MAX_SCALE, scale));
      const next = {
        x: (rect.width - layout.width * clamped) / 2,
        y: (rect.height - layout.height * clamped) / 2,
        scale: clamped,
      };
      if (smooth) withAnimation(() => setTransform(next));
      else setTransform(next);
    },
    [layout.height, layout.width, withAnimation],
  );

  useEffect(() => {
    fitToView(true);
  }, [fitToView]);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const observer = new ResizeObserver(() => fitToView(false));
    observer.observe(el);
    return () => observer.disconnect();
  }, [fitToView]);

  useEffect(() => {
    return () => {
      if (inertiaFrame.current !== null) cancelAnimationFrame(inertiaFrame.current);
    };
  }, []);

  const clientToCanvas = useCallback(
    (clientX: number, clientY: number) => {
      const el = containerRef.current;
      if (!el) return { x: 0, y: 0 };
      const rect = el.getBoundingClientRect();
      return {
        x: (clientX - rect.left - transform.x) / transform.scale,
        y: (clientY - rect.top - transform.y) / transform.scale,
      };
    },
    [transform],
  );

  const nodeAnchor = useCallback(
    (personId: string) => {
      const node = layout.nodes.find((n) => n.person.id === personId);
      if (!node) return null;
      return {
        cx: node.x + CANVAS_PADDING + NODE_WIDTH / 2,
        cy: node.y + CANVAS_PADDING + NODE_HEIGHT / 2,
        bottom: node.y + CANVAS_PADDING + NODE_HEIGHT,
      };
    },
    [layout.nodes],
  );

  const personIdAtPoint = (clientX: number, clientY: number) => {
    const el = document.elementFromPoint(clientX, clientY);
    const node = el?.closest("[data-tree-node]") as HTMLElement | null;
    return node?.dataset.personId ?? null;
  };

  const stopInertia = () => {
    if (inertiaFrame.current !== null) {
      cancelAnimationFrame(inertiaFrame.current);
      inertiaFrame.current = null;
    }
  };

  const startInertia = () => {
    stopInertia();
    const step = () => {
      velocity.current.x *= 0.9;
      velocity.current.y *= 0.9;
      if (Math.abs(velocity.current.x) < 0.15 && Math.abs(velocity.current.y) < 0.15) {
        inertiaFrame.current = null;
        return;
      }
      setTransform((t) => ({
        ...t,
        x: t.x + velocity.current.x,
        y: t.y + velocity.current.y,
      }));
      inertiaFrame.current = requestAnimationFrame(step);
    };
    inertiaFrame.current = requestAnimationFrame(step);
  };

  const zoomAt = (clientX: number, clientY: number, delta: number) => {
    const el = containerRef.current;
    if (!el) return;
    const rect = el.getBoundingClientRect();
    const px = clientX - rect.left;
    const py = clientY - rect.top;

    setTransform((t) => {
      const nextScale = Math.min(MAX_SCALE, Math.max(MIN_SCALE, t.scale * delta));
      const ratio = nextScale / t.scale;
      return {
        scale: nextScale,
        x: px - (px - t.x) * ratio,
        y: py - (py - t.y) * ratio,
      };
    });
  };

  const panBy = (dx: number, dy: number) => {
    withAnimation(() => {
      setTransform((t) => ({ ...t, x: t.x + dx, y: t.y + dy }));
    });
  };

  const onWheel = (e: React.WheelEvent) => {
    if (linkDrag || linkMenu) return;
    e.preventDefault();
    const delta = e.deltaY < 0 ? 1.07 : 1 / 1.07;
    zoomAt(e.clientX, e.clientY, delta);
  };

  const onPointerDown = (e: React.PointerEvent) => {
    if (marriedInMenu) setMarriedInMenu(null);
    if (linkDrag || linkMenu) return;
    if (e.button !== 0) return;
    if ((e.target as HTMLElement).closest("[data-tree-node]")) return;
    if ((e.target as HTMLElement).closest("[data-link-handle]")) return;
    stopInertia();
    dragging.current = true;
    lastPointer.current = { x: e.clientX, y: e.clientY };
    velocity.current = { x: 0, y: 0 };
    e.currentTarget.setPointerCapture(e.pointerId);
  };

  const onPointerMove = (e: React.PointerEvent) => {
    if (linkDrag) return;
    if (!dragging.current) return;
    const dx = e.clientX - lastPointer.current.x;
    const dy = e.clientY - lastPointer.current.y;
    velocity.current = { x: dx, y: dy };
    lastPointer.current = { x: e.clientX, y: e.clientY };
    setTransform((t) => ({ ...t, x: t.x + dx, y: t.y + dy }));
  };

  const onPointerUp = (e: React.PointerEvent) => {
    if (linkDrag) return;
    if (!dragging.current) return;
    dragging.current = false;
    e.currentTarget.releasePointerCapture(e.pointerId);
    startInertia();
  };

  const startLinkDrag = (personId: string, e: React.PointerEvent) => {
    const anchor = nodeAnchor(personId);
    if (!anchor) return;
    stopInertia();
    dragging.current = false;
    setLinkMenu(null);
    setLinkError(null);
    const point = clientToCanvas(e.clientX, e.clientY);
    setLinkDrag({
      fromId: personId,
      x1: anchor.cx,
      y1: anchor.bottom,
      x2: point.x,
      y2: point.y,
    });
    (e.target as HTMLElement).setPointerCapture(e.pointerId);
  };

  useEffect(() => {
    if (!linkDrag) return;

    const onMove = (e: PointerEvent) => {
      const point = clientToCanvas(e.clientX, e.clientY);
      setLinkDrag((drag) => (drag ? { ...drag, x2: point.x, y2: point.y } : null));
      const hoverId = personIdAtPoint(e.clientX, e.clientY);
      setLinkHoverId(hoverId && hoverId !== linkDrag.fromId ? hoverId : null);
    };

    const onUp = (e: PointerEvent) => {
      const targetId = personIdAtPoint(e.clientX, e.clientY);
      setLinkDrag(null);
      setLinkHoverId(null);

      if (!targetId || targetId === linkDrag.fromId) return;

      // Dropping on a node fires a click; skip opening the person sheet.
      suppressNodeClickRef.current = true;

      const el = containerRef.current;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      const menuX = Math.min(Math.max(e.clientX - rect.left, 12), rect.width - 240);
      const menuY = Math.min(Math.max(e.clientY - rect.top, 12), rect.height - 180);
      setLinkMenu({ fromId: linkDrag.fromId, toId: targetId, x: menuX, y: menuY });
      setLinkError(null);
    };

    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
    window.addEventListener("pointercancel", onUp);
    return () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
      window.removeEventListener("pointercancel", onUp);
    };
  }, [linkDrag, clientToCanvas]);

  const createLink = async (type: "spouse" | "child" | "divorced") => {
    if (!linkMenu) return;
    setLinkBusy(true);
    setLinkError(null);
    try {
      if (type === "spouse") {
        await api.createRelationship(familyId, linkMenu.fromId, linkMenu.toId, "spouse");
      } else if (type === "divorced") {
        await api.createRelationship(familyId, linkMenu.fromId, linkMenu.toId, "spouse", {
          marital_status: "divorced",
        });
      } else {
        await api.createRelationship(familyId, linkMenu.toId, linkMenu.fromId, "parent");
      }
      setLinkMenu(null);
      await onRelationshipsChanged();
    } catch (err) {
      setLinkError(formatLinkError(err));
    } finally {
      setLinkBusy(false);
    }
  };

  const openMarriedInMenu = async (person: Person, clientX: number, clientY: number) => {
    const el = containerRef.current;
    if (!el) return;
    const rect = el.getBoundingClientRect();
    const menuX = Math.min(Math.max(clientX - rect.left, 12), rect.width - 240);
    const menuY = Math.min(Math.max(clientY - rect.top, 12), rect.height - 180);
    setMarriedInLoading(true);
    setMarriedInMenu({ person, families: [], x: menuX, y: menuY });
    try {
      const families = await api.personFamilies(person.id);
      const otherFamilies = families.filter((f) => f.id !== familyId);
      if (otherFamilies.length === 0) {
        setMarriedInMenu(null);
        return;
      }
      setMarriedInMenu({ person, families: otherFamilies, x: menuX, y: menuY });
    } catch {
      setMarriedInMenu(null);
    } finally {
      setMarriedInLoading(false);
    }
  };

  if (persons.length === 0) {
    return (
      <div
        className={`relative flex min-h-0 flex-1 flex-col overflow-hidden rounded-2xl border border-[#c5d0dc] shadow-[0_8px_32px_rgba(30,45,60,0.08)] ${className}`}
      >
        <div className="flex flex-1 items-center justify-center bg-[#e8eef4] p-8">
          {canEdit ? (
            <div className="relative w-full max-w-sm">
              <p className="mb-4 text-center text-sm text-[#5c6b78]">
                Add your first family member to start the tree.
              </p>
              <TreeAddPersonPanel
                familyId={familyId}
                persons={persons}
                variant="inline"
                onClose={() => undefined}
                onAdded={onRelationshipsChanged}
              />
            </div>
          ) : (
            <p className="text-center text-[#8a8278]">No people in this family tree yet.</p>
          )}
        </div>
      </div>
    );
  }

  const menuFrom = linkMenu ? personById.get(linkMenu.fromId) : null;
  const menuTo = linkMenu ? personById.get(linkMenu.toId) : null;

  const spouseLinkBlocked = useMemo(() => {
    if (!linkMenu || !menuFrom || !menuTo) return { disabled: false, reason: null as string | null };
    if (alreadyLinkedAsSpouses(linkMenu.fromId, linkMenu.toId, relationships)) {
      return { disabled: true, reason: "These people are already linked as spouses." };
    }
    if (!isAvailableSpousePartner(linkMenu.toId, linkMenu.fromId, relationships, menuTo)) {
      if (menuFrom.has_spouse) {
        return {
          disabled: true,
          reason: `${menuFrom.given_name} is already married${menuFrom.spouse_name ? ` to ${menuFrom.spouse_name}` : ""}.`,
        };
      }
      if (menuTo.has_spouse) {
        return {
          disabled: true,
          reason: `${menuTo.given_name} is already married${menuTo.spouse_name ? ` to ${menuTo.spouse_name}` : ""}.`,
        };
      }
      return { disabled: true, reason: "One of these people already has a spouse." };
    }
    return { disabled: false, reason: null };
  }, [linkMenu, menuFrom, menuTo, relationships]);

  const divorcedLinkBlocked = useMemo(() => {
    if (!linkMenu) return { disabled: false, reason: null as string | null };
    if (alreadyLinkedAsSpouses(linkMenu.fromId, linkMenu.toId, relationships)) {
      return { disabled: true, reason: "These people are already linked as spouses." };
    }
    return { disabled: false, reason: null };
  }, [linkMenu, relationships]);

  return (
    <div
      className={`relative flex min-h-0 flex-1 flex-col overflow-hidden rounded-2xl border border-[#c5d0dc] shadow-[0_8px_32px_rgba(30,45,60,0.08)] ${className}`}
    >
      {canEdit && (
        <div className="flex shrink-0 items-center justify-between gap-3 border-b border-[#c5d0dc] bg-white/80 px-4 py-2 text-xs text-[#5c6b78] backdrop-blur-sm">
          <p>
            Drag the <span className="font-medium text-[#1e2a36]">+</span> on a person to link them as spouse or
            child.
          </p>
          <button
            type="button"
            onClick={() => {
              setAddPersonAnchorId(null);
              setAddPersonOpen(true);
              setLinkMenu(null);
              setMarriedInMenu(null);
            }}
            className="shrink-0 rounded-lg bg-[#4f86ad] px-3 py-1.5 text-xs font-medium text-white hover:bg-[#3d6f94]"
          >
            + Add member
          </button>
        </div>
      )}
      <div
        ref={containerRef}
        className="relative min-h-0 flex-1 cursor-grab overflow-hidden active:cursor-grabbing"
        style={{
          touchAction: "none",
          backgroundColor: "#dce4ec",
          backgroundImage:
            "radial-gradient(circle at 1px 1px, rgba(120, 140, 160, 0.22) 1px, transparent 0), linear-gradient(180deg, #e8eef4 0%, #d4dde8 100%)",
          backgroundSize: "28px 28px, 100% 100%",
        }}
        onWheel={onWheel}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onPointerCancel={onPointerUp}
      >
        <div
          className="absolute left-0 top-0 origin-top-left"
          style={{
            width: layout.width,
            height: layout.height,
            transform: `translate(${transform.x}px, ${transform.y}px) scale(${transform.scale})`,
            transition: animating ? "transform 280ms cubic-bezier(0.22, 1, 0.36, 1)" : "none",
            willChange: "transform",
          }}
        >
          <div className="relative" style={{ width: layout.width, height: layout.height }}>
            {layout.coupleBoxes.map((box, i) => (
              <div
                key={`couple-${i}`}
                className="pointer-events-none absolute z-0 rounded-[20px] border border-[#b8c8d8] bg-white/85 shadow-[0_4px_16px_rgba(30,45,60,0.1)]"
                style={{
                  left: box.x + CANVAS_PADDING,
                  top: box.y + CANVAS_PADDING,
                  width: box.width,
                  height: box.height,
                }}
              />
            ))}

            <svg
              className="pointer-events-none absolute inset-0 z-[1]"
              width={layout.width}
              height={layout.height}
              aria-hidden
            >
              {layout.edges.map((edge, i) => (
                <line
                  key={i}
                  x1={edge.x1 + CANVAS_PADDING}
                  y1={edge.y1 + CANVAS_PADDING}
                  x2={edge.x2 + CANVAS_PADDING}
                  y2={edge.y2 + CANVAS_PADDING}
                  stroke={
                    edge.kind === "spouse"
                      ? "#c94d6f"
                      : edge.kind === "sibling"
                        ? "#6a9a88"
                        : "#8fa3b3"
                  }
                  strokeWidth={edge.kind === "spouse" ? 2.5 : 1.75}
                  opacity={edge.kind === "parent" ? 0.9 : edge.dashed ? 0.65 : 1}
                  strokeDasharray={edge.dashed ? "8 5" : undefined}
                  strokeLinecap="round"
                />
              ))}
              {linkDrag && (
                <line
                  x1={linkDrag.x1}
                  y1={linkDrag.y1}
                  x2={linkDrag.x2}
                  y2={linkDrag.y2}
                  stroke="#c94d6f"
                  strokeWidth={2}
                  strokeDasharray="6 4"
                  strokeLinecap="round"
                />
              )}
            </svg>

            {layout.nodes.map(({ person, x, y }) => (
              <div
                key={person.id}
                className="absolute z-[2]"
                style={{
                  left: x + CANVAS_PADDING,
                  top: y + CANVAS_PADDING,
                  width: NODE_WIDTH,
                  height: NODE_HEIGHT,
                }}
              >
                <TreePersonNode
                  person={person}
                  canEdit={canEdit}
                  highlight={
                    linkDrag?.fromId === person.id
                      ? "source"
                      : linkHoverId === person.id
                        ? "target"
                        : null
                  }
                  onLinkDragStart={
                    canEdit ? (e) => startLinkDrag(person.id, e) : undefined
                  }
                  onClick={() => {
                    if (suppressNodeClickRef.current) {
                      suppressNodeClickRef.current = false;
                      return;
                    }
                    if (linkMenu || linkDrag || marriedInMenu || addPersonOpen) return;
                    onSelect(person);
                  }}
                  onContextMenu={
                    person.married_in
                      ? (e) => {
                          e.preventDefault();
                          e.stopPropagation();
                          setLinkMenu(null);
                          void openMarriedInMenu(person, e.clientX, e.clientY);
                        }
                      : undefined
                  }
                />
              </div>
            ))}
          </div>
        </div>

        {linkMenu && menuFrom && menuTo && (
          <TreeLinkMenu
            from={menuFrom}
            to={menuTo}
            x={linkMenu.x}
            y={linkMenu.y}
            busy={linkBusy}
            error={linkError}
            spouseDisabled={spouseLinkBlocked.disabled}
            spouseDisabledReason={spouseLinkBlocked.reason}
            divorcedDisabled={divorcedLinkBlocked.disabled}
            divorcedDisabledReason={divorcedLinkBlocked.reason}
            onSpouse={() => void createLink("spouse")}
            onDivorced={() => void createLink("divorced")}
            onChild={() => void createLink("child")}
            onCancel={() => {
              setLinkMenu(null);
              setLinkError(null);
            }}
          />
        )}

        {addPersonOpen && canEdit && (
          <TreeAddPersonPanel
            familyId={familyId}
            persons={persons}
            anchorPersonId={addPersonAnchorId}
            onClose={() => {
              setAddPersonOpen(false);
              setAddPersonAnchorId(null);
            }}
            onAdded={async () => {
              setAddPersonOpen(false);
              setAddPersonAnchorId(null);
              await onRelationshipsChanged();
            }}
          />
        )}

        {marriedInMenu && (
          <div
            className="pointer-events-auto absolute z-30 w-56 rounded-xl border border-[#e2d4f4] bg-white p-3 shadow-[0_8px_30px_rgba(90,60,120,0.18)]"
            style={{ left: marriedInMenu.x, top: marriedInMenu.y }}
            onPointerDown={(e) => e.stopPropagation()}
          >
            <p className="mb-2 text-xs text-[#8f6bab]">
              Open family tree for{" "}
              <span className="font-medium text-[#5c3d6e]">{marriedInMenu.person.given_name}</span>
            </p>
            {marriedInLoading ? (
              <p className="text-xs text-[#8a8278]">Loading families…</p>
            ) : (
              <div className="flex flex-col gap-1">
                {marriedInMenu.families.map((family) => (
                  <button
                    key={family.id}
                    type="button"
                    onClick={() => {
                      setMarriedInMenu(null);
                      navigate(`/families/${family.id}`);
                    }}
                    className="rounded-lg px-3 py-2 text-left text-sm font-medium text-[#5c3d6e] hover:bg-[#f5f0fa]"
                  >
                    {family.name}
                  </button>
                ))}
                <button
                  type="button"
                  onClick={() => setMarriedInMenu(null)}
                  className="rounded-lg px-3 py-1.5 text-sm text-[#8a8278] hover:bg-[#f3efe8]"
                >
                  Cancel
                </button>
              </div>
            )}
          </div>
        )}

        <div className="pointer-events-none absolute inset-x-0 bottom-4 flex justify-center">
          <div className="pointer-events-auto flex items-center gap-1 rounded-full bg-white/95 px-2 py-1.5 shadow-[0_4px_20px_rgba(30,45,60,0.16)] ring-1 ring-[#c5d0dc] backdrop-blur-sm">
            <NavButton label="Pan up" onClick={() => panBy(0, 80)}>
              <path d="M8 11 4 7h8L8 11z" />
            </NavButton>
            <NavButton label="Pan down" onClick={() => panBy(0, -80)}>
              <path d="M8 5 12 9H4L8 5z" />
            </NavButton>
            <span className="mx-0.5 h-5 w-px bg-[#e8e2d8]" />
            <NavButton label="Pan left" onClick={() => panBy(80, 0)}>
              <path d="M5 8 9 4v8L5 8z" />
            </NavButton>
            <NavButton label="Pan right" onClick={() => panBy(-80, 0)}>
              <path d="M11 8 7 12V4l4 4z" />
            </NavButton>
            <span className="mx-0.5 h-5 w-px bg-[#e8e2d8]" />
            <NavButton label="Fit tree to view" onClick={() => fitToView(true)}>
              <path d="M3 4h3v2H5v7H3V4zm10 0h-3v2h1v7h2V4zM8 6a3 3 0 1 0 0 6 3 3 0 0 0 0-6zm0 1.2a1.8 1.8 0 1 1 0 3.6 1.8 1.8 0 0 1 0-3.6z" />
            </NavButton>
            <NavButton
              label="Zoom in"
              onClick={() => {
                const rect = containerRef.current?.getBoundingClientRect();
                if (rect) zoomAt(rect.left + rect.width / 2, rect.top + rect.height / 2, 1.2);
              }}
            >
              <path d="M7 3a4 4 0 1 0 2.83 6.83l2.1 2.1 1.07-1.06-2.1-2.1A4 4 0 0 0 7 3zm0 1.5a2.5 2.5 0 1 1 0 5 2.5 2.5 0 0 1 0-5z" />
            </NavButton>
          </div>
        </div>
      </div>
    </div>
  );
}

function NavButton({
  children,
  label,
  onClick,
}: {
  children: React.ReactNode;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      onClick={onClick}
      className="flex h-8 w-8 items-center justify-center rounded-full text-[#5c6b78] transition hover:bg-[#eef2f6] hover:text-[#1e2a36]"
    >
      <svg viewBox="0 0 16 16" className="h-4 w-4 fill-current" aria-hidden>
        {children}
      </svg>
    </button>
  );
}