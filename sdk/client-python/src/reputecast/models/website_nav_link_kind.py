from enum import StrEnum

class WebsiteNavLinkKind(StrEnum):
    ARCHIVE = "archive"
    HOME = "home"
    PAGE = "page"
    TAG = "tag"
    URL = "url"

    def __str__(self) -> str:
        return str(self.value)
