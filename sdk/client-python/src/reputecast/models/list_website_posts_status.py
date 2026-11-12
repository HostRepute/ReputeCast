from enum import StrEnum

class ListWebsitePostsStatus(StrEnum):
    ALL = "all"
    DRAFT = "draft"
    PUBLISHED = "published"
    SCHEDULED = "scheduled"
    UNPUBLISHED = "unpublished"
    VALUE_5 = ""

    def __str__(self) -> str:
        return str(self.value)
