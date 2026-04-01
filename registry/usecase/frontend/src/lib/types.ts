export interface User {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  org_name?: string;
  created_at: string;
  updated_at: string;
}

export interface UseCase {
  id: string;
  title: string;
  slug: string;
  description: string;
  domain: string;
  industry_tags?: string;
  risk_level: string;
  roi_potential: string;
  status: string;
  impl_status: string;
  visibility: string;
  safety_pii: boolean;
  safety_autonomous: boolean;
  safety_blast_radius?: string;
  safety_oversight?: string;
  forked_from?: string;
  fork_count: number;
  author_id: string;
  author?: User;
  created_at: string;
  updated_at: string;
}

export interface UseCaseListResult {
  use_cases: UseCase[];
  total: number;
  page: number;
  limit: number;
}

export interface DashboardResult {
  use_cases: UseCase[];
}

export const DOMAINS = [
  "Customer Support",
  "Finance",
  "Productivity",
  "Trust & Safety",
  "Sales",
  "Healthcare",
  "Engineering",
  "Legal",
  "Marketing",
  "HR",
  "Operations",
] as const;

export const RISK_LEVELS = ["low", "medium", "high"] as const;
export const ROI_LEVELS = ["low", "medium", "high"] as const;
export const IMPL_STATUSES = [
  "evaluating",
  "pilot",
  "production",
  "deprecated",
] as const;
