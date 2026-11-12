from enum import StrEnum

class TransactionalChecksInputEditorMode(StrEnum):
    HTML = "html"
    MJML = "mjml"
    VISUAL = "visual"

    def __str__(self) -> str:
        return str(self.value)
