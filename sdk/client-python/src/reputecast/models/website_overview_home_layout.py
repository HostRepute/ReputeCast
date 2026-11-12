from enum import StrEnum

class WebsiteOverviewHomeLayout(StrEnum):
    FEATURED = "featured"
    GRID = "grid"
    LIST = "list"

    def __str__(self) -> str:
        return str(self.value)
