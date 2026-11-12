from enum import StrEnum

class ImportDetailStep(StrEnum):
    CONSENT = "consent"
    DONE = "done"
    FIELDS = "fields"
    IMPORT = "import"
    PREVIEW = "preview"
    SOURCE = "source"

    def __str__(self) -> str:
        return str(self.value)
