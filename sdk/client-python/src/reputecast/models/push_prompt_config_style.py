from enum import StrEnum

class PushPromptConfigStyle(StrEnum):
    BAR = "bar"
    BELL = "bell"
    DIALOG = "dialog"

    def __str__(self) -> str:
        return str(self.value)
