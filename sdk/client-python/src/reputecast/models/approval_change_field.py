from enum import StrEnum

class ApprovalChangeField(StrEnum):
    AUDIENCE = "audience"
    AUDIENCE_COUNT = "audience_count"
    CONTENT = "content"
    LANGUAGE = "language"
    LINK = "link"
    PREHEADER = "preheader"
    ROUTE = "route"
    SCHEDULE = "schedule"
    SENDER = "sender"
    SUBJECT = "subject"

    def __str__(self) -> str:
        return str(self.value)
