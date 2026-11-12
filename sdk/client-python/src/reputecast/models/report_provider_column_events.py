from enum import StrEnum

class ReportProviderColumnEvents(StrEnum):
    BOUNCE_MAILBOX = "bounce_mailbox"
    NONE = "none"
    SNS = "sns"
    WEBHOOK = "webhook"

    def __str__(self) -> str:
        return str(self.value)
