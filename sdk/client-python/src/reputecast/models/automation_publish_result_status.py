from enum import StrEnum

class AutomationPublishResultStatus(StrEnum):
    APPROVAL_REQUIRED = "approval_required"
    PUBLISHED = "published"

    def __str__(self) -> str:
        return str(self.value)
