from enum import StrEnum

class DigestDetailMode(StrEnum):
    AUTO = "auto"
    DRAFT = "draft"

    def __str__(self) -> str:
        return str(self.value)
