from enum import StrEnum

class DigestIssueInfoMode(StrEnum):
    AUTO = "auto"
    DRAFT = "draft"

    def __str__(self) -> str:
        return str(self.value)
