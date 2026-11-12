from enum import StrEnum

class TrafficActionKind(StrEnum):
    CANARY = "canary"
    COMPLAINT_WATCH = "complaint_watch"
    REPLY_RULE = "reply_rule"
    THRESHOLD = "threshold"

    def __str__(self) -> str:
        return str(self.value)
