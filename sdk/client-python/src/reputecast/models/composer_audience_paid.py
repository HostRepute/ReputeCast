from enum import StrEnum

class ComposerAudiencePaid(StrEnum):
    FREE = "free"
    PAID = "paid"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
