from enum import StrEnum

class ComposerAudienceInputCadence(StrEnum):
    ANNOUNCEMENT = "announcement"
    REGULAR = "regular"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
