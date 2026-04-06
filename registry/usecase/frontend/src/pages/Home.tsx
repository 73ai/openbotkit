import { useState, useEffect, useMemo } from "react";
import Layout from "@/components/Layout";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import { apiFetch } from "@/lib/api";
import type { UseCase, UseCaseListResult } from "@/lib/types";
import { DOMAINS, RISK_LEVELS } from "@/lib/types";

const riskColors: Record<string, string> = {
  low: "bg-green-100 text-green-800",
  medium: "bg-yellow-100 text-yellow-800",
  high: "bg-red-100 text-red-800",
};

const roiColors: Record<string, string> = {
  low: "bg-gray-100 text-gray-800",
  medium: "bg-blue-100 text-blue-800",
  high: "bg-purple-100 text-purple-800",
};

function UseCaseRow({ uc }: { uc: UseCase }) {
  return (
    <a href={`/usecase.html?id=${uc.id}`} className="block">
      <Card className="hover:shadow-md transition-shadow cursor-pointer">
        <CardHeader>
          <div className="flex items-center justify-between gap-4">
            <div className="flex-1 min-w-0">
              <CardTitle className="text-lg mb-1">{uc.title}</CardTitle>
              <CardDescription className="line-clamp-2">
                {uc.description}
              </CardDescription>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <Badge variant="outline" className={riskColors[uc.risk_level]}>
                {uc.risk_level} risk
              </Badge>
              <Badge variant="outline" className={roiColors[uc.roi_potential]}>
                {uc.roi_potential} ROI
              </Badge>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3 text-xs text-muted-foreground">
            {uc.industry_tags && (
              <span>{uc.industry_tags.split(",").join(" / ")}</span>
            )}
            {uc.fork_count > 0 && (
              <span>
                {uc.fork_count} fork{uc.fork_count !== 1 ? "s" : ""}
              </span>
            )}
            <span>by {uc.author?.name || "Unknown"}</span>
          </div>
        </CardContent>
      </Card>
    </a>
  );
}

export default function Home() {
  const [useCases, setUseCases] = useState<UseCase[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState("");
  const [domain, setDomain] = useState("");
  const [risk, setRisk] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    setLoading(true);
    setError("");
    const params = new URLSearchParams();
    if (query) params.set("q", query);
    if (domain) params.set("domain", domain);
    if (risk) params.set("risk", risk);
    params.set("limit", "50");

    apiFetch<UseCaseListResult>(`/api/usecases?${params}`)
      .then((r) => {
        setUseCases(r.use_cases || []);
        setTotal(r.total);
      })
      .catch((e) => {
        setUseCases([]);
        setError(e.message || "Failed to load use cases. Please try again.");
      })
      .finally(() => setLoading(false));
  }, [query, domain, risk]);

  const grouped = useMemo(() => {
    const map = new Map<string, UseCase[]>();
    for (const uc of useCases) {
      const list = map.get(uc.domain) || [];
      list.push(uc);
      map.set(uc.domain, list);
    }
    return map;
  }, [useCases]);

  return (
    <Layout>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">AI Use Case Registry</h1>
          <p className="text-muted-foreground mt-1">
            Discover where to apply AI in your organization. Fork use cases to
            customize for your team.
          </p>
        </div>

        <div className="flex gap-3 flex-wrap">
          <Input
            placeholder="Search use cases..."
            className="max-w-xs"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <Select
            value={domain}
            onValueChange={(v) => setDomain(v === "all" ? "" : v)}
          >
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="All Domains" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Domains</SelectItem>
              {DOMAINS.map((d) => (
                <SelectItem key={d} value={d}>
                  {d}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select
            value={risk}
            onValueChange={(v) => setRisk(v === "all" ? "" : v)}
          >
            <SelectTrigger className="w-[150px]">
              <SelectValue placeholder="All Risk Levels" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Risk Levels</SelectItem>
              {RISK_LEVELS.map((r) => (
                <SelectItem key={r} value={r}>
                  {r.charAt(0).toUpperCase() + r.slice(1)} Risk
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {error ? (
          <p className="text-muted-foreground">{error}</p>
        ) : loading ? (
          <p className="text-muted-foreground">Loading use cases...</p>
        ) : useCases.length === 0 ? (
          <div className="text-muted-foreground space-y-1">
            <p>No use cases found.</p>
            {(query || domain || risk) && (
              <p className="text-sm">
                Try adjusting your search or filters.
              </p>
            )}
          </div>
        ) : (
          <>
            <p className="text-sm text-muted-foreground">
              {total} use case{total !== 1 ? "s" : ""}
            </p>
            <div className="space-y-10 max-w-3xl">
              {Array.from(grouped.entries()).map(([domainName, ucs]) => (
                <section key={domainName}>
                  <h2 className="text-xl font-semibold mb-4 pb-2 border-b">
                    {domainName}
                  </h2>
                  <div className="space-y-3">
                    {ucs.map((uc) => (
                      <UseCaseRow key={uc.id} uc={uc} />
                    ))}
                  </div>
                </section>
              ))}
            </div>
          </>
        )}
      </div>
    </Layout>
  );
}
