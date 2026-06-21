import type { Person, Relationship } from "../types";
import { isActiveSpouseRelationship } from "./relationshipMetadata";

export const NODE_WIDTH = 112;
export const NODE_HEIGHT = 132;
export const H_GAP = 56;
export const V_GAP = 96;
export const SPOUSE_GAP = 16;
export const SIBLING_GAP = 36;
export const COUPLE_BOX_PAD = 10;
export const SIBLING_RAIL_OFFSET = 18;
export const LAYOUT_PADDING = 72;

export type PositionedNode = {
  person: Person;
  x: number;
  y: number;
};

export type PositionedEdge = {
  x1: number;
  y1: number;
  x2: number;
  y2: number;
  kind: "parent" | "spouse" | "sibling";
  dashed?: boolean;
};

export type PositionedCoupleBox = {
  x: number;
  y: number;
  width: number;
  height: number;
  personIds: string[];
};

export type TreeLayout = {
  nodes: PositionedNode[];
  edges: PositionedEdge[];
  coupleBoxes: PositionedCoupleBox[];
  width: number;
  height: number;
};

type Maps = {
  byId: Map<string, Person>;
  parentsOf: Map<string, string[]>;
  childrenOf: Map<string, string[]>;
  spousesOf: Map<string, string[]>;
};

function buildMaps(persons: Person[], relationships: Relationship[]): Maps {
  const byId = new Map(persons.map((p) => [p.id, p]));
  const parentsOf = new Map<string, string[]>();
  const childrenOf = new Map<string, string[]>();
  const spousesOf = new Map<string, string[]>();

  const push = (map: Map<string, string[]>, key: string, value: string) => {
    const list = map.get(key) || [];
    if (!list.includes(value)) list.push(value);
    map.set(key, list);
  };

  for (const rel of relationships) {
    if (rel.type === "parent") {
      push(parentsOf, rel.from_person_id, rel.to_person_id);
      push(childrenOf, rel.to_person_id, rel.from_person_id);
    } else if (rel.type === "spouse") {
      push(spousesOf, rel.from_person_id, rel.to_person_id);
      push(spousesOf, rel.to_person_id, rel.from_person_id);
    }
  }

  return { byId, parentsOf, childrenOf, spousesOf };
}

function parentKey(parentIds: string[]) {
  if (parentIds.length === 0) return "";
  return [...parentIds].sort().join(":");
}

function parentEdgeKey(childId: string, parentId: string) {
  return `${childId}\0${parentId}`;
}

/** Parent links that close a cycle (e.g. A→B and B→A) blow up generation assignment. */
function findCyclicParentEdges(
  personIds: string[],
  parentsOf: Map<string, string[]>,
): Set<string> {
  const cyclic = new Set<string>();

  const canReach = (from: string, target: string, visited: Set<string>): boolean => {
    if (from === target) return true;
    if (visited.has(from)) return false;
    visited.add(from);
    for (const parentId of parentsOf.get(from) || []) {
      if (canReach(parentId, target, visited)) return true;
    }
    return false;
  };

  for (const childId of personIds) {
    for (const parentId of parentsOf.get(childId) || []) {
      if (canReach(parentId, childId, new Set())) {
        cyclic.add(parentEdgeKey(childId, parentId));
      }
    }
  }

  return cyclic;
}

function assignGenerations(
  personIds: string[],
  parentsOf: Map<string, string[]>,
  spousesOf: Map<string, string[]>,
) {
  const gen = new Map<string, number>();
  const cyclicParentEdges = findCyclicParentEdges(personIds, parentsOf);

  for (const id of personIds) {
    if (!(parentsOf.get(id)?.length)) gen.set(id, 0);
  }
  if (gen.size === 0) {
    for (const id of personIds) gen.set(id, 0);
  }

  const maxPasses = Math.max(personIds.length, 1) + 2;
  for (let pass = 0; pass < maxPasses; pass++) {
    let changed = false;
    for (const id of personIds) {
      for (const parentId of parentsOf.get(id) || []) {
        if (cyclicParentEdges.has(parentEdgeKey(id, parentId))) continue;
        const next = (gen.get(parentId) ?? 0) + 1;
        if ((gen.get(id) ?? 0) < next) {
          gen.set(id, next);
          changed = true;
        }
      }
      for (const spouseId of spousesOf.get(id) || []) {
        const shared = Math.max(gen.get(id) ?? 0, gen.get(spouseId) ?? 0);
        if ((gen.get(id) ?? 0) < shared) {
          gen.set(id, shared);
          changed = true;
        }
        if ((gen.get(spouseId) ?? 0) < shared) {
          gen.set(spouseId, shared);
          changed = true;
        }
      }
    }
    if (!changed) break;
  }
  return gen;
}

function spousePairKey(a: string, b: string) {
  return [a, b].sort().join(":");
}

function buildClusters(
  personIds: string[],
  gen: Map<string, number>,
  relationships: Relationship[],
) {
  const byGen = new Map<number, string[][]>();
  const placed = new Set<string>();

  const spouseRels = relationships
    .filter((rel) => rel.type === "spouse")
    .sort((a, b) => {
      const aActive = isActiveSpouseRelationship(a) ? 0 : 1;
      const bActive = isActiveSpouseRelationship(b) ? 0 : 1;
      return aActive - bActive;
    });

  for (const rel of spouseRels) {
    const a = rel.from_person_id;
    const b = rel.to_person_id;
    if (placed.has(a) || placed.has(b)) continue;
    if ((gen.get(a) ?? 0) !== (gen.get(b) ?? 0)) continue;
    const cluster = [a, b].sort();
    placed.add(a);
    placed.add(b);
    const g = gen.get(a) ?? 0;
    byGen.set(g, [...(byGen.get(g) || []), cluster]);
  }

  const sorted = [...personIds].sort((x, y) => (gen.get(x) ?? 0) - (gen.get(y) ?? 0));
  for (const id of sorted) {
    if (placed.has(id)) continue;
    const g = gen.get(id) ?? 0;
    byGen.set(g, [...(byGen.get(g) || []), [id]]);
    placed.add(id);
  }

  return byGen;
}

function divorcedSpousePairKeys(relationships: Relationship[]) {
  const keys = new Set<string>();
  for (const rel of relationships) {
    if (rel.type === "spouse" && !isActiveSpouseRelationship(rel)) {
      keys.add(spousePairKey(rel.from_person_id, rel.to_person_id));
    }
  }
  return keys;
}

function clusterWidth(cluster: string[]) {
  return cluster.length * NODE_WIDTH + (cluster.length - 1) * SPOUSE_GAP;
}

function setClusterCenter(cluster: string[], centerX: number, positions: Map<string, number>) {
  const width = clusterWidth(cluster);
  let offset = centerX - width / 2 + NODE_WIDTH / 2;
  for (const id of cluster) {
    positions.set(id, offset);
    offset += NODE_WIDTH + SPOUSE_GAP;
  }
}

function clusterBounds(cluster: string[], positions: Map<string, number>) {
  const centers = cluster.map((id) => positions.get(id)).filter((x): x is number => x !== undefined);
  if (centers.length === 0) return null;
  const left = Math.min(...centers) - NODE_WIDTH / 2;
  const right = Math.max(...centers) + NODE_WIDTH / 2;
  return { left, right, center: (left + right) / 2 };
}

function groupClustersByParentKey(clusters: string[][], parentsOf: Map<string, string[]>) {
  const batches = new Map<string, string[][]>();
  const batchKeys: string[] = [];
  for (const cluster of clusters) {
    const key = parentKey(parentsOf.get(cluster[0]) || []) || `root:${cluster[0]}`;
    if (!batches.has(key)) {
      batches.set(key, []);
      batchKeys.push(key);
    }
    batches.get(key)!.push(cluster);
  }
  batchKeys.sort((a, b) => a.localeCompare(b));
  return batchKeys.map((key) => batches.get(key)!);
}

function directChildMembers(
  parentCluster: string[],
  childGen: number,
  byGen: Map<number, string[][]>,
  parentsOf: Map<string, string[]>,
) {
  const pk = parentKey(parentCluster);
  const members = new Set<string>();
  for (const cluster of byGen.get(childGen) || []) {
    if (!cluster.some((id) => parentKey(parentsOf.get(id) || []) === pk)) continue;
    for (const id of cluster) members.add(id);
  }
  return members;
}

function membersBounds(memberIds: Iterable<string>, positions: Map<string, number>) {
  let left = Infinity;
  let right = -Infinity;
  for (const id of memberIds) {
    const cx = positions.get(id);
    if (cx === undefined) continue;
    left = Math.min(left, cx - NODE_WIDTH / 2);
    right = Math.max(right, cx + NODE_WIDTH / 2);
  }
  if (!Number.isFinite(left)) return null;
  return { left, right, center: (left + right) / 2 };
}

function childrenBoundsForParents(
  parentCluster: string[],
  childGen: number,
  byGen: Map<number, string[][]>,
  parentsOf: Map<string, string[]>,
  positions: Map<string, number>,
) {
  const members = directChildMembers(parentCluster, childGen, byGen, parentsOf);
  if (members.size === 0) return null;
  return membersBounds(members, positions);
}

function shiftClusterOnly(
  cluster: string[],
  deltaX: number,
  positions: Map<string, number>,
) {
  for (const id of cluster) {
    const cx = positions.get(id);
    if (cx !== undefined) positions.set(id, cx + deltaX);
  }
}

function childClustersForParents(
  parentCluster: string[],
  childGen: number,
  byGen: Map<number, string[][]>,
  parentsOf: Map<string, string[]>,
) {
  const pk = parentKey(parentCluster);
  return (byGen.get(childGen) || []).filter((cluster) =>
    cluster.some((id) => parentKey(parentsOf.get(id) || []) === pk),
  );
}

/** Lay out direct children as a sibling row centered under their parents. */
function layoutSiblingRowUnderParents(
  parentCluster: string[],
  childGen: number,
  byGen: Map<number, string[][]>,
  parentsOf: Map<string, string[]>,
  positions: Map<string, number>,
) {
  const childClusters = childClustersForParents(parentCluster, childGen, byGen, parentsOf);
  if (childClusters.length === 0) return;

  const parentBounds = clusterBounds(parentCluster, positions);
  if (!parentBounds) return;

  const batchWidth = childClusters.reduce(
    (sum, cluster, i) => sum + clusterWidth(cluster) + (i > 0 ? SIBLING_GAP : 0),
    0,
  );
  let cursor = parentBounds.center - batchWidth / 2;
  for (let i = 0; i < childClusters.length; i++) {
    const cluster = childClusters[i];
    const center = cursor + clusterWidth(cluster) / 2;
    setClusterCenter(cluster, center, positions);
    cursor += clusterWidth(cluster) + (i < childClusters.length - 1 ? SIBLING_GAP : 0);
  }
}

/** Top-down: sibling rows under parents, then snap parents to the row center. */
function layoutFamiliesTopDown(
  byGen: Map<number, string[][]>,
  parentsOf: Map<string, string[]>,
  childrenOf: Map<string, string[]>,
  generations: number[],
  positions: Map<string, number>,
) {
  for (let gi = 0; gi < generations.length - 1; gi++) {
    const g = generations[gi];
    const childGen = generations[gi + 1];
    const parentClusters = byGen.get(g) || [];

    for (const parentCluster of parentClusters) {
      layoutSiblingRowUnderParents(parentCluster, childGen, byGen, parentsOf, positions);
    }

    const childClusters = byGen.get(childGen) || [];
    if (childClusters.length > 1) {
      resolveGenerationOverlaps(childClusters, childrenOf, positions);
    }

    for (const parentCluster of parentClusters) {
      const childBounds = childrenBoundsForParents(
        parentCluster,
        childGen,
        byGen,
        parentsOf,
        positions,
      );
      const parentBounds = clusterBounds(parentCluster, positions);
      if (!childBounds || !parentBounds) continue;
      const delta = childBounds.center - parentBounds.center;
      if (Math.abs(delta) > 0.5) {
        shiftClusterOnly(parentCluster, delta, positions);
      }
    }

    resolveGenerationOverlaps(parentClusters, childrenOf, positions);
  }
}

function collectDescendants(rootIds: string[], childrenOf: Map<string, string[]>) {
  const result = new Set(rootIds);
  const queue = [...rootIds];
  while (queue.length > 0) {
    const id = queue.shift()!;
    for (const childId of childrenOf.get(id) || []) {
      if (!result.has(childId)) {
        result.add(childId);
        queue.push(childId);
      }
    }
  }
  return result;
}

function shiftSubtree(
  rootCluster: string[],
  deltaX: number,
  childrenOf: Map<string, string[]>,
  positions: Map<string, number>,
) {
  for (const id of collectDescendants(rootCluster, childrenOf)) {
    const cx = positions.get(id);
    if (cx !== undefined) positions.set(id, cx + deltaX);
  }
}

function resolveGenerationOverlaps(
  clusters: string[][],
  childrenOf: Map<string, string[]>,
  positions: Map<string, number>,
) {
  const sorted = [...clusters].sort((a, b) => {
    const aBounds = clusterBounds(a, positions);
    const bBounds = clusterBounds(b, positions);
    return (aBounds?.left ?? 0) - (bBounds?.left ?? 0);
  });

  for (let i = 1; i < sorted.length; i++) {
    const prev = clusterBounds(sorted[i - 1], positions);
    const curr = clusterBounds(sorted[i], positions);
    if (!prev || !curr) continue;
    const gap = H_GAP / 2;
    if (curr.left < prev.right + gap) {
      const delta = prev.right + gap - curr.left;
      shiftSubtree(sorted[i], delta, childrenOf, positions);
    }
  }
}

/** Bottom-up placement: children first, then parents centered over their kids. */
function assignHorizontalPositions(
  byGen: Map<number, string[][]>,
  parentsOf: Map<string, string[]>,
  childrenOf: Map<string, string[]>,
  generations: number[],
) {
  const positions = new Map<string, number>();
  if (generations.length === 0) return positions;

  const maxGen = generations[generations.length - 1];
  const leafClusters = byGen.get(maxGen) || [];
  const leafBatches = groupClustersByParentKey(leafClusters, parentsOf);

  let cursor = LAYOUT_PADDING;
  for (const batch of leafBatches) {
    for (let i = 0; i < batch.length; i++) {
      const cluster = batch[i];
      const center = cursor + clusterWidth(cluster) / 2;
      setClusterCenter(cluster, center, positions);
      cursor += clusterWidth(cluster) + (i < batch.length - 1 ? SIBLING_GAP : 0);
    }
    cursor += H_GAP;
  }

  for (let gi = generations.length - 2; gi >= 0; gi--) {
    const g = generations[gi];
    const childGen = generations[gi + 1];
    const clusters = byGen.get(g) || [];

    const anchored: string[][] = [];
    const floating: string[][] = [];

    for (const cluster of clusters) {
      const childBounds = childrenBoundsForParents(cluster, childGen, byGen, parentsOf, positions);
      if (childBounds) {
        setClusterCenter(cluster, childBounds.center, positions);
        anchored.push(cluster);
      } else {
        floating.push(cluster);
      }
    }

    resolveGenerationOverlaps(anchored, childrenOf, positions);

    let floatCursor = LAYOUT_PADDING;
    const anchoredBounds = anchored
      .map((cluster) => clusterBounds(cluster, positions))
      .filter((b): b is NonNullable<typeof b> => Boolean(b));
    if (anchoredBounds.length > 0) {
      floatCursor = Math.max(...anchoredBounds.map((b) => b.right)) + H_GAP;
    }

    for (const cluster of floating) {
      const center = floatCursor + clusterWidth(cluster) / 2;
      setClusterCenter(cluster, center, positions);
      floatCursor += clusterWidth(cluster) + H_GAP;
    }
  }

  return positions;
}

function buildCoupleBoxes(nodes: PositionedNode[], byGen: Map<number, string[][]>): PositionedCoupleBox[] {
  const nodeById = new Map(nodes.map((n) => [n.person.id, n]));
  const boxes: PositionedCoupleBox[] = [];

  for (const clusters of byGen.values()) {
    for (const memberIds of clusters) {
      if (memberIds.length < 2) continue;
      const clusterNodes = memberIds
        .map((id) => nodeById.get(id))
        .filter((n): n is PositionedNode => Boolean(n));
      if (clusterNodes.length < 2) continue;

      const sorted = [...clusterNodes].sort((a, b) => a.x - b.x);
      const left = sorted[0];
      const right = sorted[sorted.length - 1];
      const pad = COUPLE_BOX_PAD;
      boxes.push({
        x: left.x - pad,
        y: left.y - pad,
        width: right.x + NODE_WIDTH - left.x + pad * 2,
        height: NODE_HEIGHT + pad * 2,
        personIds: memberIds,
      });
    }
  }

  return boxes;
}

function coupleBoxForParents(parentIds: string[], boxes: PositionedCoupleBox[]) {
  if (parentIds.length < 2) return null;
  const sorted = [...parentIds].sort();
  return (
    boxes.find((box) => {
      const boxSorted = [...box.personIds].sort();
      return (
        boxSorted.length === sorted.length && boxSorted.every((id, i) => id === sorted[i])
      );
    }) ?? null
  );
}

export function computeTreeLayout(persons: Person[], relationships: Relationship[]): TreeLayout {
  if (persons.length === 0) {
    return { nodes: [], edges: [], coupleBoxes: [], width: 0, height: 0 };
  }

  const { byId, parentsOf, childrenOf, spousesOf } = buildMaps(persons, relationships);
  const personIds = persons.map((p) => p.id);
  const gen = assignGenerations(personIds, parentsOf, spousesOf);
  const byGen = buildClusters(personIds, gen, relationships);
  const divorcedPairs = divorcedSpousePairKeys(relationships);
  const generations = [...byGen.keys()].sort((a, b) => a - b);
  const positions = assignHorizontalPositions(byGen, parentsOf, childrenOf, generations);
  layoutFamiliesTopDown(byGen, parentsOf, childrenOf, generations, positions);

  for (const g of generations) {
    const clusters = byGen.get(g) || [];
    if (clusters.length > 1) {
      resolveGenerationOverlaps(clusters, childrenOf, positions);
    }
  }

  const minGen = generations[0] ?? 0;
  const nodes: PositionedNode[] = [];
  for (const id of personIds) {
    const person = byId.get(id);
    const x = positions.get(id);
    const g = gen.get(id) ?? 0;
    if (!person || x === undefined) continue;
    nodes.push({
      person,
      x: x - NODE_WIDTH / 2,
      y: (g - minGen) * (NODE_HEIGHT + V_GAP),
    });
  }

  const coupleBoxes = buildCoupleBoxes(nodes, byGen);

  const edges: PositionedEdge[] = [];
  const anchorOf = (id: string) => {
    const node = nodes.find((n) => n.person.id === id);
    if (!node) return null;
    return {
      cx: node.x + NODE_WIDTH / 2,
      top: node.y,
      bottom: node.y + NODE_HEIGHT,
      midY: node.y + NODE_HEIGHT / 2,
    };
  };

  const addConnector = (
    x1: number,
    y1: number,
    x2: number,
    y2: number,
    kind: PositionedEdge["kind"],
    dashed = false,
  ) => {
    edges.push({ x1, y1, x2, y2, kind, dashed: dashed || undefined });
  };

  const addParentChildConnector = (
    parentCx: number,
    parentBottom: number,
    childCx: number,
    childTop: number,
  ) => {
    const midY = parentBottom + (childTop - parentBottom) * 0.55;
    addConnector(parentCx, parentBottom, parentCx, midY, "parent");
    addConnector(parentCx, midY, childCx, midY, "parent");
    addConnector(childCx, midY, childCx, childTop, "parent");
  };

  for (const box of coupleBoxes) {
    const anchors = box.personIds
      .map((id) => anchorOf(id))
      .filter((a): a is NonNullable<ReturnType<typeof anchorOf>> => Boolean(a))
      .sort((a, b) => a.cx - b.cx);
    const divorced = divorcedPairs.has(spousePairKey(...box.personIds));
    for (let i = 0; i < anchors.length - 1; i++) {
      const left = anchors[i];
      const right = anchors[i + 1];
      addConnector(
        left.cx + NODE_WIDTH / 2,
        left.midY,
        right.cx - NODE_WIDTH / 2,
        right.midY,
        "spouse",
        divorced,
      );
    }
  }

  const parentsByChild = new Map<string, string[]>();
  for (const rel of relationships) {
    if (rel.type !== "parent") continue;
    const list = parentsByChild.get(rel.from_person_id) || [];
    if (!list.includes(rel.to_person_id)) list.push(rel.to_person_id);
    parentsByChild.set(rel.from_person_id, list);
  }

  const childrenByParents = new Map<string, string[]>();
  for (const [childId, parentIds] of parentsByChild) {
    const key = parentKey(parentIds);
    if (!key) continue;
    const list = childrenByParents.get(key) || [];
    if (!list.includes(childId)) list.push(childId);
    childrenByParents.set(key, list);
  }

  const handledChildren = new Set<string>();

  for (const [key, childIds] of childrenByParents) {
    const parentIds = key.split(":").filter(Boolean);
    const childAnchors = childIds
      .map((id) => anchorOf(id))
      .filter((a): a is NonNullable<ReturnType<typeof anchorOf>> => Boolean(a));

    if (childAnchors.length === 0) continue;

    const sameRow = childAnchors.every(
      (a) => Math.abs(a.top - childAnchors[0].top) < 6,
    );
    if (!sameRow) continue;

    const sorted = [...childAnchors].sort((a, b) => a.cx - b.cx);
    const railY = sorted[0].top - SIBLING_RAIL_OFFSET;
    const railLeft = sorted[0].cx;
    const railRight = sorted[sorted.length - 1].cx;
    const railCx = (railLeft + railRight) / 2;

    if (sorted.length >= 2) {
      addConnector(railLeft, railY, railRight, railY, "sibling");
      for (const child of sorted) {
        addConnector(child.cx, railY, child.cx, child.top, "parent");
      }
      for (const id of childIds) handledChildren.add(id);
    }

    const box = coupleBoxForParents(parentIds, coupleBoxes);
    if (sorted.length >= 2) {
      const fromBottom = box ? box.y + box.height : anchorOf(parentIds[0])?.bottom;
      const fromCx = box ? box.x + box.width / 2 : anchorOf(parentIds[0])?.cx;
      if (fromBottom !== undefined && fromCx !== undefined) {
        const midY = fromBottom + (railY - fromBottom) * 0.55;
        addConnector(fromCx, fromBottom, fromCx, midY, "parent");
        addConnector(fromCx, midY, railCx, midY, "parent");
        addConnector(railCx, midY, railCx, railY, "parent");
      }
    } else if (sorted.length === 1) {
      const child = sorted[0];
      const childId = childIds[0];
      if (box && parentIds.length >= 2) {
        addParentChildConnector(box.x + box.width / 2, box.y + box.height, child.cx, child.top);
      } else {
        for (const parentId of parentIds) {
          const parent = anchorOf(parentId);
          if (parent) addParentChildConnector(parent.cx, parent.bottom, child.cx, child.top);
        }
      }
      handledChildren.add(childId);
    }
  }

  for (const [childId, parentIds] of parentsByChild) {
    if (handledChildren.has(childId)) continue;
    const child = anchorOf(childId);
    if (!child) continue;
    const box = coupleBoxForParents(parentIds, coupleBoxes);
    if (box && parentIds.length >= 2) {
      addParentChildConnector(box.x + box.width / 2, box.y + box.height, child.cx, child.top);
    } else {
      for (const parentId of parentIds) {
        const parent = anchorOf(parentId);
        if (parent) addParentChildConnector(parent.cx, parent.bottom, child.cx, child.top);
      }
    }
  }

  for (const rel of relationships) {
    if (rel.type !== "spouse") continue;
    if (rel.from_person_id > rel.to_person_id) continue;
    const divorced = !isActiveSpouseRelationship(rel);
    const key = [rel.from_person_id, rel.to_person_id].sort().join(":");
    const inBox = coupleBoxes.some((box) => {
      const boxKey = [...box.personIds].sort().join(":");
      return boxKey === key;
    });
    if (inBox) continue;

    const a = anchorOf(rel.from_person_id);
    const b = anchorOf(rel.to_person_id);
    if (!a || !b) continue;
    if (Math.abs(a.midY - b.midY) <= 6) {
      const left = a.cx < b.cx ? a : b;
      const right = a.cx < b.cx ? b : a;
      addConnector(
        left.cx + NODE_WIDTH / 2,
        left.midY,
        right.cx - NODE_WIDTH / 2,
        right.midY,
        "spouse",
        divorced,
      );
    } else {
      const top = a.midY < b.midY ? a : b;
      const bottom = a.midY < b.midY ? b : a;
      const midY = top.midY + (bottom.midY - top.midY) * 0.5;
      addConnector(top.cx, top.midY, top.cx, midY, "spouse", divorced);
      addConnector(top.cx, midY, bottom.cx, midY, "spouse", divorced);
      addConnector(bottom.cx, midY, bottom.cx, bottom.midY, "spouse", divorced);
    }
  }

  const minX = Math.min(
    ...nodes.map((n) => n.x),
    ...coupleBoxes.map((b) => b.x),
    0,
  );
  const maxX = Math.max(
    nodes.reduce((m, n) => Math.max(m, n.x + NODE_WIDTH), 0),
    coupleBoxes.reduce((m, b) => Math.max(m, b.x + b.width), 0),
  );
  const maxY = Math.max(
    nodes.reduce((m, n) => Math.max(m, n.y + NODE_HEIGHT), 0),
    coupleBoxes.reduce((m, b) => Math.max(m, b.y + b.height), 0),
  );

  const contentWidth = maxX - minX;
  const shiftX = LAYOUT_PADDING - minX;

  for (const node of nodes) node.x += shiftX;
  for (const box of coupleBoxes) box.x += shiftX;
  for (const edge of edges) {
    edge.x1 += shiftX;
    edge.x2 += shiftX;
  }

  return {
    nodes,
    edges,
    coupleBoxes,
    width: contentWidth + LAYOUT_PADDING * 2,
    height: maxY + LAYOUT_PADDING * 2,
  };
}