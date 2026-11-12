from enum import StrEnum

class WebsitePublishStateRule(StrEnum):
    AUTHOR_LACKS_SEND = "author_lacks_send"
    POST_PUBLISH = "post_publish"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
