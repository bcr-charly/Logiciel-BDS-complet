export function formatDate(d: Date) {
  return new Intl.DateTimeFormat("fr-FR", { dateStyle: "long", timeStyle: "short" }).format(d);
}

export function formatDateCourt(d: Date) {
  return new Intl.DateTimeFormat("fr-FR", { dateStyle: "long" }).format(d);
}

export function formatPrix(p: number) {
  return new Intl.NumberFormat("fr-FR", { style: "currency", currency: "EUR" }).format(p);
}
