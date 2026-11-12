from enum import StrEnum

class WebsitePublishPastInputVisibility(StrEnum):
    AUDIENCE = "audience"
    MEMBERS = "members"
    PUBLIC = "public"

    def __str__(self) -> str:
        return str(self.value)
