from enum import StrEnum

class EmailLanguageStatus(StrEnum):
    DRAFT = "draft"
    MISSING = "missing"
    OUTDATED = "outdated"
    SOURCE = "source"
    UP_TO_DATE = "up_to_date"

    def __str__(self) -> str:
        return str(self.value)
