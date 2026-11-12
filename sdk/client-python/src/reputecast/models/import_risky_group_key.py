from enum import StrEnum

class ImportRiskyGroupKey(StrEnum):
    ABUSE = "abuse"
    CATCH_ALL = "catch_all"
    DISPOSABLE = "disposable"
    ROLE = "role"
    STALE = "stale"
    TRAP = "trap"
    TYPO = "typo"
    UNKNOWN = "unknown"

    def __str__(self) -> str:
        return str(self.value)
