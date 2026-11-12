from enum import StrEnum

class ContentCheckIssueGroup(StrEnum):
    COMPLIANCE = "Compliance"
    CONTENT = "Content"
    LINKS = "Links"
    PERSONALIZATION = "Personalization"
    SUBJECT = "Subject"

    def __str__(self) -> str:
        return str(self.value)
