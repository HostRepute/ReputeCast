from enum import StrEnum

class ComposerABTestMetric(StrEnum):
    CLICKS = "clicks"
    HUMAN_OPENS = "human_opens"
    MANUAL = "manual"

    def __str__(self) -> str:
        return str(self.value)
