const STATUS_CLASS: Record<string, string> = {
  done: 'badge ok',
  running: 'badge running',
  failed: 'badge bad',
  unknown: 'badge',
}

export function StatusBadge({ status }: { status: string }) {
  return <span className={STATUS_CLASS[status] ?? 'badge'}>{status}</span>
}
