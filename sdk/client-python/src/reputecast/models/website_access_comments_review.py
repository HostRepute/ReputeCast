from enum import StrEnum

class WebsiteAccessCommentsReview(StrEnum):
    ALL = "all"
    FIRST = "first"
    NONE = "none"

    def __str__(self) -> str:
        return str(self.value)
