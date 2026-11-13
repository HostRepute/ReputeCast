from enum import StrEnum

class SignupFormSourcesReferrer(StrEnum):
    ANY = "any"
    DIRECT = "direct"
    OTHER_SITE = "other_site"
    SEARCH = "search"
    SOCIAL = "social"

    def __str__(self) -> str:
        return str(self.value)
