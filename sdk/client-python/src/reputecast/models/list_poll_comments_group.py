from enum import StrEnum

class ListPollCommentsGroup(StrEnum):
    ALL = "all"
    DETRACTORS = "detractors"
    PASSIVES = "passives"
    PROMOTERS = "promoters"

    def __str__(self) -> str:
        return str(self.value)
