from enum import StrEnum

class ProductDetailStatus(StrEnum):
    ARCHIVED = "archived"
    DRAFT = "draft"
    PUBLISHED = "published"

    def __str__(self) -> str:
        return str(self.value)
