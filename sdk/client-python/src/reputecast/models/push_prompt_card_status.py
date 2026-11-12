from enum import StrEnum

class PushPromptCardStatus(StrEnum):
    DRAFT = "draft"
    LIVE = "live"
    PAUSED = "paused"

    def __str__(self) -> str:
        return str(self.value)
