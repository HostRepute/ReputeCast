from enum import StrEnum

class ComposerABTestNoWinner(StrEnum):
    DEFAULT = "default"

    def __str__(self) -> str:
        return str(self.value)
