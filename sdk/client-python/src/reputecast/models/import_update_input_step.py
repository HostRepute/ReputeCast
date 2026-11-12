from enum import StrEnum

class ImportUpdateInputStep(StrEnum):
    CONSENT = "consent"
    FIELDS = "fields"
    PREVIEW = "preview"
    SOURCE = "source"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)
