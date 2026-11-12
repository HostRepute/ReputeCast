from enum import StrEnum

class ScoringFormOptionKind(StrEnum):
    EMBEDDED = "embedded"
    HOSTED = "hosted"
    POPUP = "popup"

    def __str__(self) -> str:
        return str(self.value)
