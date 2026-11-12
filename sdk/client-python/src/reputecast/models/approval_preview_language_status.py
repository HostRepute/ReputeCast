from enum import StrEnum

class ApprovalPreviewLanguageStatus(StrEnum):
    DRAFT = "draft"
    SOURCE = "source"
    TRANSLATED = "translated"

    def __str__(self) -> str:
        return str(self.value)
