import { useCallback } from "react";
import { api } from "../../lib.api";
import { useAdminStore } from "../../store/admin.store";
import { useUiStore } from "../../store/ui.store";
import { asMessage } from "../shared/errors";

export function useAdminCenter() {
  const {
    loadState,
    error,
    users,
    onlineUsers,
    recentActivities,
    setLoadState,
    setError,
    setOverview,
  } = useAdminStore();
  const { setManagedUsers } = useUiStore();

  const refresh = useCallback(async () => {
    setLoadState("loading");
    setError("");
    try {
      const overview = await api.getAdminOverview();
      setOverview(overview);
      setManagedUsers(overview.users);
      setLoadState("ready");
    } catch (err) {
      setError(asMessage(err));
      setLoadState("error");
    }
  }, [setError, setLoadState, setManagedUsers, setOverview]);

  return {
    loadState,
    error,
    users,
    onlineUsers,
    recentActivities,
    refresh,
  };
}
