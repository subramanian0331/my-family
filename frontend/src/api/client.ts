const TOKEN_KEY = "family_tree_token";

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers);
  const token = getToken();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  if (options.body && !(options.body instanceof FormData)) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(path, { ...options, headers });
  if (!response.ok) {
    const payload = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(payload.error || "request failed");
  }
  if (response.status === 204) {
    return undefined as T;
  }
  const contentType = response.headers.get("Content-Type") || "";
  if (contentType.includes("application/json")) {
    return response.json();
  }
  return response.blob() as T;
}

async function requestArray<T>(path: string, options: RequestInit = {}): Promise<T[]> {
  const data = await request<T[] | null>(path, options);
  return data ?? [];
}

export const api = {
  me: () => request<import("../types").User>("/api/auth/me"),
  families: () => requestArray<import("../types").FamilySummary>("/api/families"),
  createFamily: (name: string, description: string) =>
    request("/api/families", {
      method: "POST",
      body: JSON.stringify({ name, description }),
    }),
  family: (id: string) => request<import("../types").FamilyDetail>(`/api/families/${id}`),
  people: (familyId: string) =>
    requestArray<import("../types").Person>(`/api/families/${familyId}/people`),
  tree: async (id: string) => {
    const data = await request<import("../types").TreeData>(`/api/families/${id}/tree`);
    return {
      persons: data?.persons ?? [],
      relationships: data?.relationships ?? [],
    };
  },
  search: (id: string, q: string) =>
    requestArray<import("../types").Person>(`/api/families/${id}/search?q=${encodeURIComponent(q)}`),
  searchPeople: (q: string, familyId?: string) => {
    const params = new URLSearchParams({ q });
    if (familyId) params.set("family_id", familyId);
    return requestArray<import("../types").PersonSearchHit>(`/api/people/search?${params}`);
  },
  addPersonToFamily: (familyId: string, personId: string) =>
    request<import("../types").Person>(`/api/families/${familyId}/people/${personId}/add`, {
      method: "POST",
    }),
  removePersonFromFamily: (familyId: string, personId: string) =>
    request(`/api/families/${familyId}/people/${personId}`, { method: "DELETE" }),
  setFamilyMarriageLabel: (familyId: string, personId: string, marriedIn: boolean) =>
    requestArray<import("../types").PersonFamilyRef>(
      `/api/families/${familyId}/people/${personId}/family-label`,
      { method: "PATCH", body: JSON.stringify({ married_in: marriedIn }) },
    ),
  personFamilies: (personId: string) =>
    requestArray<import("../types").PersonFamilyRef>(`/api/people/${personId}/families`),
  createPerson: (familyId: string, person: Partial<import("../types").Person>) =>
    request<import("../types").Person>(`/api/families/${familyId}/people`, {
      method: "POST",
      body: JSON.stringify(person),
    }),
  bulkCreatePeople: (
    familyId: string,
    payload: import("../types").BulkCreatePayload,
  ) =>
    request<{ people: import("../types").Person[]; count: number }>(
      `/api/families/${familyId}/people/bulk`,
      { method: "POST", body: JSON.stringify(payload) },
    ),
  updatePerson: (personId: string, familyId: string, person: Partial<import("../types").Person>) =>
    request<import("../types").Person>(`/api/people/${personId}?family_id=${familyId}`, {
      method: "PATCH",
      body: JSON.stringify(person),
    }),
  createRelationship: (
    familyId: string,
    fromPersonId: string,
    toPersonId: string,
    type: "parent" | "spouse",
    metadata?: import("../types").RelationshipMetadata,
  ) =>
    request(`/api/families/${familyId}/relationships`, {
      method: "POST",
      body: JSON.stringify({
        from_person_id: fromPersonId,
        to_person_id: toPersonId,
        type,
        ...(metadata ? { metadata } : {}),
      }),
    }),
  updateRelationship: (
    relationshipId: string,
    familyId: string,
    metadata: import("../types").RelationshipMetadata,
  ) =>
    request(`/api/relationships/${relationshipId}?family_id=${familyId}`, {
      method: "PATCH",
      body: JSON.stringify({ metadata }),
    }),
  deleteRelationship: (relationshipId: string, familyId: string) =>
    request(`/api/relationships/${relationshipId}?family_id=${familyId}`, {
      method: "DELETE",
    }),
  uploadPhoto: async (personId: string, familyId: string, file: File) => {
    const form = new FormData();
    form.append("photo", file);
    return request(`/api/people/${personId}/photos?family_id=${familyId}`, {
      method: "POST",
      body: form,
    });
  },
  photoUrl: (photoId: string) => `/api/photos/${photoId}`,
  pendingInvites: () => requestArray<import("../types").Invite>("/api/invites/pending"),
  acceptInvite: (token: string) =>
    request("/api/invites/accept", { method: "POST", body: JSON.stringify({ token }) }),
  createInvite: (familyId: string, email: string, role: string) =>
    request(`/api/families/${familyId}/invites`, {
      method: "POST",
      body: JSON.stringify({ email, role }),
    }),
  members: (familyId: string) =>
    requestArray<import("../types").FamilyMember>(`/api/families/${familyId}/members`),
  exportGedcom: async (familyId: string) => {
    const token = getToken();
    const response = await fetch(`/api/families/${familyId}/gedcom/export`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    });
    if (!response.ok) throw new Error("export failed");
    return response.blob();
  },
  importGedcom: async (familyId: string, file: File) => {
    const token = getToken();
    const response = await fetch(`/api/families/${familyId}/gedcom/import`, {
      method: "POST",
      headers: {
        "Content-Type": "text/plain",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      body: await file.text(),
    });
    if (!response.ok) {
      const payload = await response.json().catch(() => ({ error: "import failed" }));
      throw new Error(payload.error);
    }
    return response.json();
  },
};