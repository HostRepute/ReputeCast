from enum import StrEnum

class WebsitePostPreviewInputAs(StrEnum):
    FREE = "free"
    FREE_USED = "free_used"
    PAID = "paid"
    VISITOR = "visitor"
    VISITOR_USED = "visitor_used"

    def __str__(self) -> str:
        return str(self.value)
