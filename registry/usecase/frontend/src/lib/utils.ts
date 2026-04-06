import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

const hexPattern = /^[a-f0-9]+$/;

export function isValidId(id: string | null): id is string {
  return !!id && hexPattern.test(id);
}
