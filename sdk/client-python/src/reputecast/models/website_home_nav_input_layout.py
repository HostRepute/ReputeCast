from enum import StrEnum

class WebsiteHomeNavInputLayout(StrEnum):
    FEATURED = "featured"
    GRID = "grid"
    LIST = "list"

    def __str__(self) -> str:
        return str(self.value)
