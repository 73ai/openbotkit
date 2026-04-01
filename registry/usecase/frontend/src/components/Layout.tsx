import type { ReactNode } from "react";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";

export default function Layout({ children }: { children: ReactNode }) {
  const { user, loading, login, logout } = useAuth();

  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b">
        <div className="max-w-6xl mx-auto px-4 h-14 flex items-center justify-between">
          <div className="flex items-center gap-6">
            <a href="/" className="font-bold text-lg">
              AI Use Case Registry
            </a>
            <nav className="flex items-center gap-4 text-sm">
              <a
                href="/"
                className="text-muted-foreground hover:text-foreground"
              >
                Browse
              </a>
              {user && (
                <a
                  href="/dashboard"
                  className="text-muted-foreground hover:text-foreground"
                >
                  Dashboard
                </a>
              )}
            </nav>
          </div>
          <div className="flex items-center gap-3">
            {loading ? null : user ? (
              <>
                <span className="text-sm text-muted-foreground">
                  {user.name}
                </span>
                <Button variant="ghost" size="sm" onClick={logout}>
                  Sign out
                </Button>
              </>
            ) : (
              <Button size="sm" onClick={login}>
                Sign in with Google
              </Button>
            )}
          </div>
        </div>
      </header>

      <main className="flex-1 max-w-6xl mx-auto px-4 py-8 w-full">
        {children}
      </main>

      <footer className="border-t py-6">
        <div className="max-w-6xl mx-auto px-4 text-center text-sm text-muted-foreground">
          Powered by{" "}
          <a
            href="https://openbotkit.dev"
            className="underline hover:text-foreground"
          >
            OpenBotKit
          </a>
        </div>
      </footer>
    </div>
  );
}
