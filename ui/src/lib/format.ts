export function formatDate(value: string) {
  return new Date(value).toLocaleDateString(undefined, {
    day: "numeric",
    month: "long",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  })
}
export function deadline(value: string) {
  const milliseconds = new Date(value).getTime() - Date.now()
  if (milliseconds <= 0) return "Expired"
  const days = Math.floor(milliseconds / 86_400_000)
  const hours = Math.floor((milliseconds % 86_400_000) / 3_600_000)
  return days ? `${days}d ${hours}h` : `${hours}h`
}
