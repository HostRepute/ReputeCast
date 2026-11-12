from enum import StrEnum

class WebsitePostRowStatus(StrEnum):
    DRAFT = "draft"
    PUBLISHED = "published"
    SCHEDULED = "scheduled"
    UNPUBLISHED = "unpublished"

    def __str__(self) -> str:
        return str(self.value)
