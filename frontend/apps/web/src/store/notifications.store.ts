import { create } from "zustand";
import type { NotificationItem } from "../lib.types";

interface NotificationsStore {
  notifications: NotificationItem[];
  setNotifications: (
    notifications: NotificationItem[] | ((current: NotificationItem[]) => NotificationItem[]),
  ) => void;
}

export const useNotificationsStore = create<NotificationsStore>((set) => ({
  notifications: [],
  setNotifications: (notifications) =>
    set((state) => ({
      notifications: typeof notifications === "function" ? notifications(state.notifications) : notifications,
    })),
}));
