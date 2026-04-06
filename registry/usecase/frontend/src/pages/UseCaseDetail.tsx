import { useState, useEffect } from "react";
import Layout from "@/components/Layout";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { useAuth } from "@/lib/auth";
import { apiFetch } from "@/lib/api";
import { isValidId } from "@/lib/utils";
import type { UseCase } from "@/lib/types";

const riskColors: Record<string, string> = {
  low: "bg-green-100 text-green-800",
  medium: "bg-yellow-100 text-yellow-800",
  high: "bg-red-100 text-red-800",
};

export default function UseCaseDetail() {
  const { user } = useAuth();
  const [uc, setUc] = useState<UseCase | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [forking, setForking] = useState(false);

  const id = new URLSearchParams(window.location.search).get("id");

  useEffect(() => {
    if (!isValidId(id)) {
      setError("Invalid use case ID.");
      setLoading(false);
      return;
    }
    apiFetch<UseCase>(`/api/usecases/${id}`)
      .then(setUc)
      .catch((e) => setError(e.message || "Failed to load use case."))
      .finally(() => setLoading(false));
  }, [id]);

  const handleFork = async () => {
    if (!uc || !user) return;
    if (!confirm("Fork this use case to your dashboard? You can customize it after.")) return;
    setForking(true);
    try {
      const fork = await apiFetch<UseCase>(`/api/usecases/${uc.id}/fork`, {
        method: "POST",
      });
      window.location.href = `/usecase.html?id=${fork.id}`;
    } catch {
      alert("Failed to fork use case. Please try again.");
    } finally {
      setForking(false);
    }
  };

  if (loading) {
    return (
      <Layout>
        <p className="text-muted-foreground">Loading use case...</p>
      </Layout>
    );
  }

  if (error || !uc) {
    return (
      <Layout>
        <div className="space-y-4">
          <p className="text-muted-foreground">
            {error || "Use case not found."}
          </p>
          <Button variant="outline" asChild>
            <a href="/">Back to browse</a>
          </Button>
        </div>
      </Layout>
    );
  }

  const isAuthor = user?.id === uc.author_id;
  const wasUpdated = uc.updated_at !== uc.created_at;

  return (
    <Layout>
      <div className="space-y-6 max-w-3xl">
        <div>
          <a
            href="/"
            className="text-sm text-muted-foreground hover:text-foreground"
          >
            Back to browse
          </a>
        </div>

        <div className="flex items-start justify-between gap-4">
          <div>
            <div className="flex items-center gap-2 flex-wrap mb-2">
              <Badge variant="secondary">{uc.domain}</Badge>
              <Badge variant="outline" className={riskColors[uc.risk_level]}>
                {uc.risk_level} risk
              </Badge>
              <Badge variant="outline">{uc.roi_potential} ROI</Badge>
              <Badge variant="outline">{uc.impl_status}</Badge>
              {uc.visibility === "private" && (
                <Badge variant="destructive">Private</Badge>
              )}
              {uc.status === "draft" && (
                <Badge variant="outline">Draft</Badge>
              )}
            </div>
            <h1 className="text-3xl font-bold">{uc.title}</h1>
            {uc.forked_from && (
              <p className="text-sm text-muted-foreground mt-1">
                Forked from{" "}
                <a
                  href={`/usecase.html?id=${uc.forked_from}`}
                  className="underline"
                >
                  original
                </a>
              </p>
            )}
          </div>
          <div className="flex gap-2 shrink-0">
            {isAuthor && (
              <Button variant="outline" asChild>
                <a href={`/usecase-form.html?id=${uc.id}`}>Edit</a>
              </Button>
            )}
            {user && !isAuthor && (
              <Button onClick={handleFork} disabled={forking}>
                {forking ? "Forking..." : `Fork (${uc.fork_count})`}
              </Button>
            )}
            {!user && (
              <Button variant="outline" disabled>
                Sign in to fork
              </Button>
            )}
          </div>
        </div>

        {uc.industry_tags && (
          <div className="flex gap-2 flex-wrap">
            {uc.industry_tags.split(",").map((tag) => (
              <Badge key={tag} variant="secondary">
                {tag.trim()}
              </Badge>
            ))}
          </div>
        )}

        <Card>
          <CardHeader>
            <CardTitle>Description</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="whitespace-pre-wrap">{uc.description}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Safety Checklist</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center gap-2">
              <Checkbox checked={uc.safety_pii} disabled />
              <Label>Handles PII (Personally Identifiable Information)</Label>
            </div>
            <div className="flex items-center gap-2">
              <Checkbox checked={uc.safety_autonomous} disabled />
              <Label>Autonomous actions (acts without human approval)</Label>
            </div>
            {uc.safety_blast_radius && (
              <div>
                <p className="text-sm font-medium mb-1">Blast Radius</p>
                <p className="text-sm text-muted-foreground">
                  {uc.safety_blast_radius}
                </p>
              </div>
            )}
            {uc.safety_oversight && (
              <div>
                <p className="text-sm font-medium mb-1">Human Oversight</p>
                <p className="text-sm text-muted-foreground">
                  {uc.safety_oversight}
                </p>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div className="text-sm text-muted-foreground">
                <span>
                  Created by {uc.author?.name || "Unknown"} on{" "}
                  {new Date(uc.created_at).toLocaleDateString()}
                </span>
                {wasUpdated && (
                  <span className="ml-3">
                    Updated {new Date(uc.updated_at).toLocaleDateString()}
                  </span>
                )}
              </div>
              <Button asChild>
                <a href="https://openbotkit.dev">
                  Implement with OpenBotKit
                </a>
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}
