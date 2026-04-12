export default function StatusBadge({ status }) {
  if (!status) return <span className="badge badge-cancelled">—</span>
  return <span className={`badge badge-${status.toLowerCase()}`}>{status}</span>
}
