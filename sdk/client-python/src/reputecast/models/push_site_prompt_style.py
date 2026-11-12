from enum import StrEnum

class PushSitePromptStyle(StrEnum):
    BAR = "bar"
    BELL = "bell"
    DIALOG = "dialog"

    def __str__(self) -> str:
        return str(self.value)
