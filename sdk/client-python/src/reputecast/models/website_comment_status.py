from enum import StrEnum

class WebsiteCommentStatus(StrEnum):
    DELETED = "deleted"
    HIDDEN = "hidden"
    PENDING = "pending"
    PUBLISHED = "published"

    def __str__(self) -> str:
        return str(self.value)
