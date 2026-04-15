export function slugify(name: string): string {
  return name
    .toLowerCase()
    .replace(/\s+/g, '-')          // whitespace runs → single dash
    .replace(/[^a-z0-9-]/g, '')    // strip everything outside [a-z0-9-]
    .replace(/-+/g, '-')           // collapse dash runs
    .replace(/^-+|-+$/g, '')       // trim leading/trailing dashes
}
