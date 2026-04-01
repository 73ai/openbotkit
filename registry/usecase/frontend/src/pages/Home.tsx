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

export default function Home() {
  const [useCases, setUseCases] = useState<UseCase[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState("");
  const [domain, setDomain] = useState("");
  const [risk, setRisk] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
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
      .catch(() => setUseCases([]))
      .finally(() => setLoading(false));
  }, [query, domain, risk]);

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
          <Select value={domain} onValueChange={(v) => setDomain(v === "all" ? "" : v)}>
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
          <Select value={risk} onValueChange={(v) => setRisk(v === "all" ? "" : v)}>
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

        {loading ? (
          <p className="text-muted-foreground">Loading...</p>
        ) : useCases.length === 0 ? (
          <p className="text-muted-foreground">No use cases found.</p>
        ) : (
          <>
            <p className="text-sm text-muted-foreground">
              {total} use case{total !== 1 ? "s" : ""}
            </p>
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {useCases.map((uc) => (
                <a key={uc.id} href={`/usecase.html?id=${uc.id}`}>
                  <Card className="h-full hover:shadow-md transition-shadow cursor-pointer">
                    <CardHeader>
                      <div className="flex items-center gap-2 flex-wrap mb-2">
                        <Badge variant="secondary">{uc.domain}</Badge>
                        <Badge
                          variant="outline"
                          className={riskColors[uc.risk_level]}
                        >
                          {uc.risk_level} risk
                        </Badge>
                      </div>
                      <CardTitle className="text-lg">{uc.title}</CardTitle>
                      <CardDescription className="line-clamp-2">
                        {uc.description}
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="flex items-center gap-2 flex-wrap">
                        <Badge
                          variant="outline"
                          className={roiColors[uc.roi_potential]}
                        >
                          {uc.roi_potential} ROI
                        </Badge>
                        {uc.fork_count > 0 && (
                          <span className="text-xs text-muted-foreground">
                            {uc.fork_count} fork
                            {uc.fork_count !== 1 ? "s" : ""}
                          </span>
                        )}
                      </div>
                    </CardContent>
                    <CardFooter>
                      <span className="text-xs text-muted-foreground">
                        by {uc.author?.name || "Unknown"}
                      </span>
                    </CardFooter>
                  </Card>
                </a>
              ))}
            </div>
          </>
        )}
      </div>
    </Layout>
  );
}
