from enum import StrEnum

class TransactionalPublishLanguageStatus(StrEnum):
    DRAFT = "draft"
    OUTDATED = "outdated"
    SOURCE = "source"
    UP_TO_DATE = "up_to_date"

    def __str__(self) -> str:
        return str(self.value)
