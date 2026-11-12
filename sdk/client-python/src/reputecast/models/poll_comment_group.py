from enum import StrEnum

class PollCommentGroup(StrEnum):
    DETRACTOR = "detractor"
    PASSIVE = "passive"
    PROMOTER = "promoter"

    def __str__(self) -> str:
        return str(self.value)
