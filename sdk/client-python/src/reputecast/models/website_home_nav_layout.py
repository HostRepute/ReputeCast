from enum import StrEnum

class WebsiteHomeNavLayout(StrEnum):
    FEATURED = "featured"
    GRID = "grid"
    LIST = "list"

    def __str__(self) -> str:
        return str(self.value)
