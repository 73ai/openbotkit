import { useState, useEffect, useCallback } from "react";
import { apiFetch } from "./api";
import type { User } from "./types";

let cachedUser: User | null = null;
let fetchPromise: Promise<User | null> | null = null;

function fetchUser(): Promise<User | null> {
  if (!fetchPromise) {
    fetchPromise = apiFetch<User>("/api/auth/me")
      .then((u) => {
        cachedUser = u;
        return u;
      })
      .catch(() => {
        cachedUser = null;
        return null;
      });
  }
  return fetchPromise;
}

export function useAuth() {
  const [user, setUser] = useState<User | null>(cachedUser);
  const [loading, setLoading] = useState(cachedUser === null && !fetchPromise);

  useEffect(() => {
    if (cachedUser) {
      setUser(cachedUser);
      setLoading(false);
      return;
    }
    setLoading(true);
    fetchUser().then((u) => {
      setUser(u);
      setLoading(false);
    });
  }, []);

  const login = useCallback(async () => {
    const { url } = await apiFetch<{ url: string }>("/api/auth/google");
    window.location.href = url;
  }, []);

  const logout = useCallback(async () => {
    try {
      await apiFetch("/api/auth/logout", { method: "POST" });
    } catch {
      // proceed with local logout even if API call fails
    }
    cachedUser = null;
    fetchPromise = null;
    setUser(null);
    window.location.href = "/";
  }, []);

  return { user, loading, login, logout };
}
