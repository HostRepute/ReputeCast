from enum import StrEnum

class LiveIncidentKind(StrEnum):
    CANARY = "canary"
    COMPLAINT_WATCH = "complaint_watch"
    IN_DOUBT = "in_doubt"
    PROVIDER_OUTAGE = "provider_outage"
    RECHECK_BLOCKED = "recheck_blocked"
    RENDER_ERROR = "render_error"
    REPLY_RULE = "reply_rule"
    THRESHOLD = "threshold"
    THROTTLED = "throttled"

    def __str__(self) -> str:
        return str(self.value)
