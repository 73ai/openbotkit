import { useState, useEffect } from "react";
import Layout from "@/components/Layout";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/lib/auth";
import { apiFetch } from "@/lib/api";
import type { UseCase, DashboardResult } from "@/lib/types";

export default function Dashboard() {
  const { user, loading: authLoading } = useAuth();
  const [useCases, setUseCases] = useState<UseCase[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (authLoading) return;
    if (!user) {
      window.location.href = "/";
      return;
    }
    apiFetch<DashboardResult>("/api/dashboard")
      .then((r) => setUseCases(r.use_cases || []))
      .catch(() => setUseCases([]))
      .finally(() => setLoading(false));
  }, [user, authLoading]);

  const handleDelete = async (id: string) => {
    if (!confirm("Delete this use case?")) return;
    try {
      await apiFetch(`/api/usecases/${id}`, { method: "DELETE" });
      setUseCases((prev) => prev.filter((uc) => uc.id !== id));
    } catch {
      alert("Failed to delete");
    }
  };

  if (authLoading || loading) {
    return (
      <Layout>
        <p className="text-muted-foreground">Loading...</p>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">Dashboard</h1>
            <p className="text-muted-foreground mt-1">
              Your use cases{user?.org_name ? ` at ${user.org_name}` : ""}
            </p>
          </div>
          <Button asChild>
            <a href="/usecase-form.html">Create New</a>
          </Button>
        </div>

        {useCases.length === 0 ? (
          <Card>
            <CardContent className="pt-6">
              <p className="text-center text-muted-foreground">
                You haven't created any use cases yet.{" "}
                <a href="/usecase-form.html" className="underline">
                  Create your first one
                </a>
                .
              </p>
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {useCases.map((uc) => (
              <Card key={uc.id} className="flex flex-col">
                <CardHeader>
                  <div className="flex items-center gap-2 flex-wrap mb-2">
                    <Badge variant="secondary">{uc.domain}</Badge>
                    <Badge variant="outline">{uc.status}</Badge>
                    <Badge variant="outline">{uc.visibility}</Badge>
                  </div>
                  <CardTitle className="text-lg">
                    <a
                      href={`/usecase.html?id=${uc.id}`}
                      className="hover:underline"
                    >
                      {uc.title}
                    </a>
                  </CardTitle>
                  <CardDescription className="line-clamp-2">
                    {uc.description}
                  </CardDescription>
                </CardHeader>
                <CardContent className="flex-1">
                  {uc.forked_from && (
                    <p className="text-xs text-muted-foreground">
                      Forked from{" "}
                      <a
                        href={`/usecase.html?id=${uc.forked_from}`}
                        className="underline"
                      >
                        original
                      </a>
                    </p>
                  )}
                </CardContent>
                <CardFooter className="gap-2">
                  <Button variant="outline" size="sm" asChild>
                    <a href={`/usecase-form.html?id=${uc.id}`}>Edit</a>
                  </Button>
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => handleDelete(uc.id)}
                  >
                    Delete
                  </Button>
                </CardFooter>
              </Card>
            ))}
          </div>
        )}
      </div>
    </Layout>
  );
}
