import { useQuery, useQueryClient } from "@tanstack/react-query";
import { type ReactNode, useCallback, useMemo } from "react";
import { api, NetScopeAPIError } from "../api/client";
import type { CurrentAccount } from "../api/types";
import { AuthContext, type AuthContextValue } from "./AuthContext";

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const current = useQuery({
    queryKey: ["account"],
    queryFn: api.currentAccount,
    retry: false,
    staleTime: 30_000,
  });

  const acceptAccount = useCallback(
    (account: CurrentAccount) => {
      window.localStorage.setItem(
        "netscope.workspace",
        account.activeWorkspace.id,
      );
      queryClient.setQueryData(["account"], account);
    },
    [queryClient],
  );

  const login = useCallback(
    async (email: string, password: string) => {
      acceptAccount(await api.login({ email, password }));
    },
    [acceptAccount],
  );

  const register = useCallback(
    async (input: {
      email: string;
      password: string;
      displayName: string;
      workspaceName: string;
    }) => {
      acceptAccount(await api.register(input));
    },
    [acceptAccount],
  );

  const logout = useCallback(async () => {
    await api.logout();
    window.localStorage.removeItem("netscope.workspace");
    queryClient.setQueryData(["account"], undefined);
    queryClient.removeQueries({
      predicate: (query) => query.queryKey[0] !== "capabilities",
    });
  }, [queryClient]);

  const selectWorkspace = useCallback(
    async (id: string) => {
      window.localStorage.setItem("netscope.workspace", id);
      await queryClient.invalidateQueries();
    },
    [queryClient],
  );

  const createWorkspace = useCallback(
    async (name: string) => {
      const workspace = await api.createWorkspace(name);
      window.localStorage.setItem("netscope.workspace", workspace.id);
      await queryClient.invalidateQueries();
    },
    [queryClient],
  );

  const account =
    current.error instanceof NetScopeAPIError &&
    current.error.code === "authentication_required"
      ? undefined
      : current.data;
  const value = useMemo<AuthContextValue>(
    () => ({
      account,
      loading: current.isPending,
      login,
      register,
      logout,
      selectWorkspace,
      createWorkspace,
    }),
    [
      account,
      current.isPending,
      login,
      register,
      logout,
      selectWorkspace,
      createWorkspace,
    ],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
