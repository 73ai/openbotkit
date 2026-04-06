import { useState, useEffect } from "react";
import Layout from "@/components/Layout";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import { useAuth } from "@/lib/auth";
import { apiFetch } from "@/lib/api";
import type { UseCase } from "@/lib/types";
import { DOMAINS, RISK_LEVELS, ROI_LEVELS, IMPL_STATUSES } from "@/lib/types";

export default function UseCaseForm() {
  const { user, loading: authLoading } = useAuth();
  const [saving, setSaving] = useState(false);
  const [existing, setExisting] = useState<UseCase | null>(null);

  const editId = new URLSearchParams(window.location.search).get("id");
  const isEdit = !!editId;

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [domain, setDomain] = useState("");
  const [industryTags, setIndustryTags] = useState("");
  const [riskLevel, setRiskLevel] = useState("medium");
  const [roiPotential, setRoiPotential] = useState("medium");
  const [status, setStatus] = useState("published");
  const [implStatus, setImplStatus] = useState("evaluating");
  const [visibility, setVisibility] = useState("public");
  const [safetyPii, setSafetyPii] = useState(false);
  const [safetyAutonomous, setSafetyAutonomous] = useState(false);
  const [safetyBlastRadius, setSafetyBlastRadius] = useState("");
  const [safetyOversight, setSafetyOversight] = useState("");

  useEffect(() => {
    if (authLoading) return;
    if (!user) {
      window.location.href = "/";
      return;
    }
    if (editId) {
      apiFetch<UseCase>(`/api/usecases/${editId}`)
        .then((uc) => {
          if (uc.author_id !== user.id) {
            alert("You can only edit your own use cases.");
            window.location.href = `/usecase.html?id=${editId}`;
            return;
          }
          setExisting(uc);
          setTitle(uc.title);
          setDescription(uc.description);
          setDomain(uc.domain);
          setIndustryTags(uc.industry_tags || "");
          setRiskLevel(uc.risk_level);
          setRoiPotential(uc.roi_potential);
          setStatus(uc.status);
          setImplStatus(uc.impl_status);
          setVisibility(uc.visibility);
          setSafetyPii(uc.safety_pii);
          setSafetyAutonomous(uc.safety_autonomous);
          setSafetyBlastRadius(uc.safety_blast_radius || "");
          setSafetyOversight(uc.safety_oversight || "");
        })
        .catch(() => {
          alert("Use case not found.");
          window.location.href = "/dashboard";
        });
    }
  }, [editId, user, authLoading]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);

    const body = {
      title,
      description,
      domain,
      industry_tags: industryTags,
      risk_level: riskLevel,
      roi_potential: roiPotential,
      status,
      impl_status: implStatus,
      visibility,
      safety_pii: safetyPii,
      safety_autonomous: safetyAutonomous,
      safety_blast_radius: safetyBlastRadius,
      safety_oversight: safetyOversight,
    };

    try {
      let result: UseCase;
      if (isEdit) {
        result = await apiFetch<UseCase>(`/api/usecases/${editId}`, {
          method: "PUT",
          body: JSON.stringify(body),
        });
      } else {
        result = await apiFetch<UseCase>("/api/usecases", {
          method: "POST",
          body: JSON.stringify(body),
        });
      }
      window.location.href = `/usecase.html?id=${result.id}`;
    } catch (err) {
      alert(`Failed to save: ${err}`);
    } finally {
      setSaving(false);
    }
  };

  if (authLoading) {
    return (
      <Layout>
        <p className="text-muted-foreground">Loading...</p>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="max-w-2xl">
        <h1 className="text-3xl font-bold mb-6">
          {isEdit ? "Edit Use Case" : "Create Use Case"}
        </h1>

        <form onSubmit={handleSubmit} className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="title">Title</Label>
                <Input
                  id="title"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="e.g. Customer Support Ticket Triage"
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Describe the use case, how AI is applied, and expected outcomes..."
                  rows={6}
                  required
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Domain</Label>
                  <Select value={domain} onValueChange={setDomain} required>
                    <SelectTrigger>
                      <SelectValue placeholder="Select domain" />
                    </SelectTrigger>
                    <SelectContent>
                      {DOMAINS.map((d) => (
                        <SelectItem key={d} value={d}>
                          {d}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="tags">Industry Tags</Label>
                  <Input
                    id="tags"
                    value={industryTags}
                    onChange={(e) => setIndustryTags(e.target.value)}
                    placeholder="SaaS, Fintech, Healthcare"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Risk Level</Label>
                  <Select value={riskLevel} onValueChange={setRiskLevel}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {RISK_LEVELS.map((r) => (
                        <SelectItem key={r} value={r}>
                          {r.charAt(0).toUpperCase() + r.slice(1)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label>ROI Potential</Label>
                  <Select value={roiPotential} onValueChange={setRoiPotential}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {ROI_LEVELS.map((r) => (
                        <SelectItem key={r} value={r}>
                          {r.charAt(0).toUpperCase() + r.slice(1)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Implementation Status</Label>
                  <Select value={implStatus} onValueChange={setImplStatus}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {IMPL_STATUSES.map((s) => (
                        <SelectItem key={s} value={s}>
                          {s.charAt(0).toUpperCase() + s.slice(1)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label>Visibility</Label>
                  <Select value={visibility} onValueChange={setVisibility}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="public">Public</SelectItem>
                      <SelectItem value="private">Private</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Safety Checklist</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center gap-2">
                <Checkbox
                  id="pii"
                  checked={safetyPii}
                  onCheckedChange={(c) => setSafetyPii(c === true)}
                />
                <Label htmlFor="pii">
                  Handles PII (Personally Identifiable Information)
                </Label>
              </div>

              <div className="flex items-center gap-2">
                <Checkbox
                  id="autonomous"
                  checked={safetyAutonomous}
                  onCheckedChange={(c) => setSafetyAutonomous(c === true)}
                />
                <Label htmlFor="autonomous">
                  Autonomous actions (acts without human approval)
                </Label>
              </div>

              <div className="space-y-2">
                <Label htmlFor="blast">Blast Radius</Label>
                <Input
                  id="blast"
                  value={safetyBlastRadius}
                  onChange={(e) => setSafetyBlastRadius(e.target.value)}
                  placeholder="What happens if it goes wrong?"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="oversight">Human Oversight</Label>
                <Input
                  id="oversight"
                  value={safetyOversight}
                  onChange={(e) => setSafetyOversight(e.target.value)}
                  placeholder="How do humans stay in the loop?"
                />
              </div>
            </CardContent>
          </Card>

          <div className="flex gap-3">
            <Button type="submit" disabled={saving}>
              {saving
                ? "Saving..."
                : isEdit
                  ? "Update Use Case"
                  : "Create Use Case"}
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() => history.back()}
            >
              Cancel
            </Button>
          </div>
        </form>
      </div>
    </Layout>
  );
}
