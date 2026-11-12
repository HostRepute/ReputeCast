from enum import StrEnum

class ProductSalesPageStatus(StrEnum):
    DEFAULT = "default"
    DRAFT = "draft"
    PUBLISHED = "published"
    SCHEDULED = "scheduled"
    UNPUBLISHED = "unpublished"

    def __str__(self) -> str:
        return str(self.value)
