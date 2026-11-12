from enum import StrEnum

class CommerceBlockMode(StrEnum):
    CARD = "card"
    CHOSEN = "chosen"
    COLLECTION = "collection"

    def __str__(self) -> str:
        return str(self.value)
