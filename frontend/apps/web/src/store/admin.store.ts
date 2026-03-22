import { create } from "zustand";
import type { AdminOverviewResponse, AuditEventItem, ManagedUserItem, OnlineUserItem } from "../lib.types";
import type { LoadState } from "./auth.store";

interface AdminStore {
  loadState: LoadState;
  error: string;
  users: ManagedUserItem[];
  onlineUsers: OnlineUserItem[];
  recentActivities: AuditEventItem[];
  setLoadState: (loadState: LoadState) => void;
  setError: (error: string) => void;
  setUsers: (users: ManagedUserItem[]) => void;
  setOnlineUsers: (onlineUsers: OnlineUserItem[]) => void;
  setRecentActivities: (recentActivities: AuditEventItem[]) => void;
  setOverview: (overview: AdminOverviewResponse) => void;
}

export const useAdminStore = create<AdminStore>((set) => ({
  loadState: "idle",
  error: "",
  users: [],
  onlineUsers: [],
  recentActivities: [],
  setLoadState: (loadState) => set({ loadState }),
  setError: (error) => set({ error }),
  setUsers: (users) => set({ users }),
  setOnlineUsers: (onlineUsers) => set({ onlineUsers }),
  setRecentActivities: (recentActivities) => set({ recentActivities }),
  setOverview: (overview) =>
    set({
      users: overview.users,
      onlineUsers: overview.onlineUsers,
      recentActivities: overview.recentActivities,
    }),
}));
