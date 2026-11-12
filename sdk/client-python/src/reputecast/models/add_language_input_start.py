from enum import StrEnum

class AddLanguageInputStart(StrEnum):
    COPY = "copy"
    LAYOUT = "layout"

    def __str__(self) -> str:
        return str(self.value)
