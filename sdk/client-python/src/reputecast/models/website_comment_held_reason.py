from enum import StrEnum

class WebsiteCommentHeldReason(StrEnum):
    BLOCKED = "blocked"
    FIRST_COMMENT = "first_comment"
    LINKS = "links"
    REVIEW_ALL = "review_all"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)
