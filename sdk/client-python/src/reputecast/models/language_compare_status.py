from enum import StrEnum

class LanguageCompareStatus(StrEnum):
    DRAFT = "draft"
    OUTDATED = "outdated"
    UP_TO_DATE = "up_to_date"

    def __str__(self) -> str:
        return str(self.value)
