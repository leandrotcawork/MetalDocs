import { useCallback, useRef } from "react";
import { api } from "../../lib.api";
import type { OperationsStreamSnapshot } from "../../api/notifications";
import type { NotificationItem } from "../../lib.types";
import { useNotificationsStore } from "../../store/notifications.store";
import { useUiStore } from "../../store/ui.store";
import { asMessage } from "../shared/errors";

export function useNotifications() {
  const { notifications, setNotifications } = useNotificationsStore();
  const { setError } = useUiStore();

  const handleMarkNotificationRead = useCallback(
    async (notificationId: string) => {
      try {
        await api.markNotificationRead(notificationId);
        setNotifications((current) =>
          current.map((item) =>
            item.id === notificationId ? { ...item, status: "READ", readAt: new Date().toISOString() } : item,
          ),
        );
      } catch (err) {
        setError(asMessage(err));
      }
    },
    [setError, setNotifications],
  );

  const lastSnapshotRef = useRef<OperationsStreamSnapshot | null>(null);
  const lastRefreshRef = useRef(0);

  const subscribeOperations = useCallback(
    (onRefresh: () => void) =>
      api.subscribeOperationsStream(
        (snapshot) => {
          const now = Date.now();
          const previous = lastSnapshotRef.current;
          const hasChanges = !previous
            || previous.pendingNotifications !== snapshot.pendingNotifications
            || previous.pendingApprovals !== snapshot.pendingApprovals
            || previous.documentsInReview !== snapshot.documentsInReview
            || previous.totalDocuments !== snapshot.totalDocuments;
          const enoughTimePassed = now - lastRefreshRef.current >= 15000;

          lastSnapshotRef.current = snapshot;
          if (!hasChanges && !enoughTimePassed) {
            return;
          }
          lastRefreshRef.current = now;
          onRefresh();
        },
        () => {
          // Stream keeps retrying in browser; UI fallback remains available.
        },
      ),
    [],
  );

  return {
    notifications,
    setNotifications,
    handleMarkNotificationRead,
    subscribeOperations,
  };
}
