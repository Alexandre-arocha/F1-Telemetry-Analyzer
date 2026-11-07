import { Badge } from "@/components/ui/badge";
import { REASONS } from "@/lib/format";

export function ReasonBadge({ reason }: { reason: string }) {
  return <Badge variant="secondary">{REASONS[reason] ?? reason}</Badge>;
}
