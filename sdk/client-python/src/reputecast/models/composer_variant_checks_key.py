from enum import StrEnum

class ComposerVariantChecksKey(StrEnum):
    B = "B"
    C = "C"

    def __str__(self) -> str:
        return str(self.value)
