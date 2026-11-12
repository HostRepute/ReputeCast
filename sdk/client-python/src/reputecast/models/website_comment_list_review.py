from enum import StrEnum

class WebsiteCommentListReview(StrEnum):
    ALL = "all"
    FIRST = "first"
    NONE = "none"

    def __str__(self) -> str:
        return str(self.value)
