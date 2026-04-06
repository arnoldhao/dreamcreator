export function formatTemplate(
  template: string,
  values: Record<string, string | number>
): string {
  return template.replace(/\{(\w+)\}/g, (match, key) => {
    const value = values[key]
    if (value === undefined || value === null) {
      return match
    }
    return String(value)
  })
}
