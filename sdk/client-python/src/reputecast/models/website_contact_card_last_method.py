from enum import StrEnum

class WebsiteContactCardLastMethod(StrEnum):
    CODE = "code"
    LINK = "link"
    SIGNUP = "signup"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
