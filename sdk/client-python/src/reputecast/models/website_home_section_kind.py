from enum import StrEnum

class WebsiteHomeSectionKind(StrEnum):
    ABOUT = "about"
    FEATURED = "featured"
    LATEST = "latest"
    SUBSCRIBE = "subscribe"
    TAG = "tag"
    TIERS = "tiers"

    def __str__(self) -> str:
        return str(self.value)
