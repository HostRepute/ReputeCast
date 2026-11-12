from enum import StrEnum

class DigestIssueInfoStatus(StrEnum):
    COLLECTING = "collecting"
    DRAFT = "draft"
    HELD = "held"
    NOT_SENT = "not_sent"
    PREPARED = "prepared"
    PREPARING = "preparing"
    SENDING = "sending"
    SENT = "sent"
    SKIPPED = "skipped"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)
