from enum import StrEnum

class CommerceBlockMode(StrEnum):
    BEST_SELLERS = "best_sellers"
    BOUGHT_TOGETHER = "bought_together"
    CARD = "card"
    CHOSEN = "chosen"
    COLLECTION = "collection"
    RECENTLY_VIEWED = "recently_viewed"
    VIEWED = "viewed"

    def __str__(self) -> str:
        return str(self.value)
