from enum import StrEnum

class ReportInsightKind(StrEnum):
    BLOCKED = "blocked"
    MAILBOX_BOUNCES = "mailbox_bounces"
    TOP_LINK = "top_link"

    def __str__(self) -> str:
        return str(self.value)
