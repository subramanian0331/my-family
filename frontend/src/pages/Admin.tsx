import { useCallback, useEffect, useState } from "react";
import { Link, Navigate } from "react-router-dom";
import { api } from "../api/client";
import { useAuth } from "../context/AuthContext";
import type {
  AdminFamily,
  AdminInviteDetail,
  AdminSettings,
  AdminUserDetail,
  FamilyRole,
  SiteRole,
} from "../types";

type Tab = "users" | "invites" | "settings";

const familyRoles: FamilyRole[] = ["owner", "editor", "viewer"];

export function Admin() {
  const { user } = useAuth();
  const [tab, setTab] = useState<Tab>("users");
  const [users, setUsers] = useState<AdminUserDetail[]>([]);
  const [invites, setInvites] = useState<AdminInviteDetail[]>([]);
  const [families, setFamilies] = useState<AdminFamily[]>([]);
  const [settings, setSettings] = useState<AdminSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteFamilyId, setInviteFamilyId] = useState("");
  const [inviteRole, setInviteRole] = useState<FamilyRole>("viewer");

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [userRows, inviteRows, familyRows, settingsRow] = await Promise.all([
        api.adminUsers(),
        api.adminInvites(),
        api.adminFamilies(),
        api.adminSettings(),
      ]);
      setUsers(userRows);
      setInvites(inviteRows);
      setFamilies(familyRows);
      setSettings(settingsRow);
      setInviteFamilyId((prev) => prev || familyRows[0]?.id || "");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load admin data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  if (!user) return null;
  if (user.site_role !== "admin") return <Navigate to="/" replace />;

  const updateSiteRole = async (userId: string, siteRole: SiteRole) => {
    setBusyId(userId);
    try {
      await api.adminUpdateUserRole(userId, siteRole);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Update failed");
    } finally {
      setBusyId(null);
    }
  };

  const updateFamilyRole = async (userId: string, familyId: string, role: FamilyRole) => {
    setBusyId(`${userId}-${familyId}`);
    try {
      await api.adminSetUserFamilyAccess(userId, familyId, role);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Update failed");
    } finally {
      setBusyId(null);
    }
  };

  const removeFamilyAccess = async (userId: string, familyId: string) => {
    if (!confirm("Remove this user's access to the family?")) return;
    setBusyId(`${userId}-${familyId}-rm`);
    try {
      await api.adminRemoveUserFamilyAccess(userId, familyId);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Remove failed");
    } finally {
      setBusyId(null);
    }
  };

  const addFamilyAccess = async (userId: string, familyId: string, role: FamilyRole) => {
    if (!familyId) return;
    setBusyId(`${userId}-add`);
    try {
      await api.adminSetUserFamilyAccess(userId, familyId, role);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Add failed");
    } finally {
      setBusyId(null);
    }
  };

  const sendInvite = async () => {
    if (!inviteEmail.trim() || !inviteFamilyId) return;
    setBusyId("invite-create");
    try {
      await api.adminCreateInvite(inviteFamilyId, inviteEmail.trim(), inviteRole);
      setInviteEmail("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Invite failed");
    } finally {
      setBusyId(null);
    }
  };

  const revokeInvite = async (inviteId: string) => {
    setBusyId(inviteId);
    try {
      await api.adminRevokeInvite(inviteId);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Revoke failed");
    } finally {
      setBusyId(null);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-3">
        <Link to="/" className="text-sm text-slate-500 hover:text-accent">
          ← Families
        </Link>
        <h1 className="flex-1 text-2xl font-semibold text-slate-900">Site administration</h1>
        <span className="rounded-full bg-violet-100 px-2.5 py-0.5 text-xs font-medium text-violet-700">
          Admin
        </span>
      </div>

      <div className="flex gap-2 border-b border-slate-200">
        {(["users", "invites", "settings"] as const).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`border-b-2 px-4 py-2 text-sm font-medium capitalize ${
              tab === t ? "border-accent text-accent" : "border-transparent text-slate-500"
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          {error}
        </div>
      )}

      {loading ? (
        <p className="text-slate-500">Loading…</p>
      ) : (
        <>
          {tab === "users" && (
            <div className="space-y-4">
              {users.map((row) => (
                <UserAdminCard
                  key={row.user.id}
                  row={row}
                  families={families}
                  busyId={busyId}
                  onSiteRoleChange={updateSiteRole}
                  onFamilyRoleChange={updateFamilyRole}
                  onRemoveFamily={removeFamilyAccess}
                  onAddFamily={addFamilyAccess}
                />
              ))}
            </div>
          )}

          {tab === "invites" && (
            <div className="space-y-6">
              <div className="rounded-2xl border border-slate-200 bg-white p-5">
                <h2 className="mb-3 font-medium text-slate-900">Send invite</h2>
                <div className="flex flex-wrap gap-2">
                  <input
                    className="min-w-[200px] flex-1 rounded-lg border border-slate-200 px-3 py-2 text-sm"
                    placeholder="Email (Google account)"
                    value={inviteEmail}
                    onChange={(e) => setInviteEmail(e.target.value)}
                  />
                  <select
                    className="rounded-lg border border-slate-200 px-3 py-2 text-sm"
                    value={inviteFamilyId}
                    onChange={(e) => setInviteFamilyId(e.target.value)}
                  >
                    {families.map((f) => (
                      <option key={f.id} value={f.id}>
                        {f.name}
                      </option>
                    ))}
                  </select>
                  <select
                    className="rounded-lg border border-slate-200 px-3 py-2 text-sm"
                    value={inviteRole}
                    onChange={(e) => setInviteRole(e.target.value as FamilyRole)}
                  >
                    {familyRoles.map((r) => (
                      <option key={r} value={r}>
                        {r}
                      </option>
                    ))}
                  </select>
                  <button
                    onClick={() => void sendInvite()}
                    disabled={busyId === "invite-create"}
                    className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
                  >
                    Send
                  </button>
                </div>
              </div>

              <div className="overflow-hidden rounded-2xl border border-slate-200 bg-white">
                <table className="w-full text-left text-sm">
                  <thead className="border-b border-slate-100 bg-slate-50 text-slate-600">
                    <tr>
                      <th className="px-4 py-3 font-medium">Email</th>
                      <th className="px-4 py-3 font-medium">Family</th>
                      <th className="px-4 py-3 font-medium">Role</th>
                      <th className="px-4 py-3 font-medium" />
                    </tr>
                  </thead>
                  <tbody>
                    {invites.length === 0 ? (
                      <tr>
                        <td colSpan={4} className="px-4 py-8 text-center text-slate-500">
                          No pending invites
                        </td>
                      </tr>
                    ) : (
                      invites.map((row) => (
                        <tr key={row.invite.id} className="border-b border-slate-50">
                          <td className="px-4 py-3">{row.invite.email}</td>
                          <td className="px-4 py-3">{row.family_name}</td>
                          <td className="px-4 py-3 capitalize">{row.invite.role}</td>
                          <td className="px-4 py-3 text-right">
                            <button
                              onClick={() => void revokeInvite(row.invite.id)}
                              disabled={busyId === row.invite.id}
                              className="text-sm text-red-600 hover:underline disabled:opacity-50"
                            >
                              Revoke
                            </button>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {tab === "settings" && settings && (
            <div className="grid gap-4 sm:grid-cols-2">
              <SettingsCard label="Site URL" value={settings.frontend_url} />
              <SettingsCard
                label="Google sign-in"
                value={settings.google_enabled ? "Enabled" : "Disabled"}
              />
              <SettingsCard label="Bootstrap admin email" value={settings.site_admin_email || "—"} />
              <SettingsCard label="Users" value={String(settings.user_count)} />
              <SettingsCard label="Families" value={String(settings.family_count)} />
              <SettingsCard label="Pending invites" value={String(settings.pending_invites)} />
              <div className="sm:col-span-2 rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
                Server URL and OAuth credentials are set in the server <code>.env</code> file. User
                roles and family access can be changed in the Users tab.
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}

function SettingsCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-slate-200 bg-white p-5">
      <div className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</div>
      <div className="mt-1 text-lg font-medium text-slate-900">{value}</div>
    </div>
  );
}

function UserAdminCard({
  row,
  families,
  busyId,
  onSiteRoleChange,
  onFamilyRoleChange,
  onRemoveFamily,
  onAddFamily,
}: {
  row: AdminUserDetail;
  families: AdminFamily[];
  busyId: string | null;
  onSiteRoleChange: (userId: string, role: SiteRole) => void;
  onFamilyRoleChange: (userId: string, familyId: string, role: FamilyRole) => void;
  onRemoveFamily: (userId: string, familyId: string) => void;
  onAddFamily: (userId: string, familyId: string, role: FamilyRole) => void;
}) {
  const [addFamilyId, setAddFamilyId] = useState("");
  const [addRole, setAddRole] = useState<FamilyRole>("viewer");
  const u = row.user;
  const memberFamilyIds = new Set(row.families.map((f) => f.family_id));
  const availableFamilies = families.filter((f) => !memberFamilyIds.has(f.id));

  return (
    <div className="rounded-2xl border border-slate-200 bg-white p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex items-center gap-3">
          {u.avatar_url ? (
            <img src={u.avatar_url} alt="" className="h-10 w-10 rounded-full" />
          ) : (
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-slate-200 text-sm font-medium">
              {u.name?.[0] || u.email[0]}
            </div>
          )}
          <div>
            <div className="font-medium text-slate-900">{u.name || u.email}</div>
            <div className="text-sm text-slate-500">{u.email}</div>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <label className="text-xs text-slate-500">Site role</label>
          <select
            className="rounded-lg border border-slate-200 px-2 py-1.5 text-sm capitalize"
            value={u.site_role}
            disabled={busyId === u.id}
            onChange={(e) => onSiteRoleChange(u.id, e.target.value as SiteRole)}
          >
            <option value="user">user</option>
            <option value="admin">admin</option>
          </select>
        </div>
      </div>

      <div className="mt-4">
        <h3 className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-500">
          Family access
        </h3>
        {row.families.length === 0 ? (
          <p className="text-sm text-slate-500">No family memberships</p>
        ) : (
          <ul className="space-y-2">
            {row.families.map((f) => (
              <li
                key={f.family_id}
                className="flex flex-wrap items-center justify-between gap-2 rounded-lg bg-slate-50 px-3 py-2"
              >
                <span className="text-sm font-medium text-slate-800">{f.family_name}</span>
                <div className="flex items-center gap-2">
                  <select
                    className="rounded border border-slate-200 px-2 py-1 text-sm capitalize"
                    value={f.role}
                    disabled={busyId === `${u.id}-${f.family_id}`}
                    onChange={(e) =>
                      onFamilyRoleChange(u.id, f.family_id, e.target.value as FamilyRole)
                    }
                  >
                    {familyRoles.map((r) => (
                      <option key={r} value={r}>
                        {r}
                      </option>
                    ))}
                  </select>
                  <button
                    type="button"
                    onClick={() => onRemoveFamily(u.id, f.family_id)}
                    disabled={busyId === `${u.id}-${f.family_id}-rm`}
                    className="text-xs text-red-600 hover:underline"
                  >
                    Remove
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}

        {availableFamilies.length > 0 && (
          <div className="mt-3 flex flex-wrap items-center gap-2">
            <select
              className="rounded-lg border border-slate-200 px-2 py-1.5 text-sm"
              value={addFamilyId}
              onChange={(e) => setAddFamilyId(e.target.value)}
            >
              <option value="">Add to family…</option>
              {availableFamilies.map((f) => (
                <option key={f.id} value={f.id}>
                  {f.name}
                </option>
              ))}
            </select>
            <select
              className="rounded-lg border border-slate-200 px-2 py-1.5 text-sm capitalize"
              value={addRole}
              onChange={(e) => setAddRole(e.target.value as FamilyRole)}
            >
              {familyRoles.map((r) => (
                <option key={r} value={r}>
                  {r}
                </option>
              ))}
            </select>
            <button
              type="button"
              disabled={!addFamilyId || busyId === `${u.id}-add`}
              onClick={() => {
                onAddFamily(u.id, addFamilyId, addRole);
                setAddFamilyId("");
              }}
              className="rounded-lg border border-slate-200 px-3 py-1.5 text-sm hover:bg-slate-50 disabled:opacity-50"
            >
              Add access
            </button>
          </div>
        )}
      </div>
    </div>
  );
}