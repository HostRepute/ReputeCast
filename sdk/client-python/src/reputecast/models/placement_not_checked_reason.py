from enum import StrEnum

class PlacementNotCheckedReason(StrEnum):
    SEED_REMOVED = "seed_removed"
    SIGNIN_EXPIRED = "signin_expired"
    SIGNIN_FAILED = "signin_failed"
    STOPPED = "stopped"
    UNSENT = "unsent"

    def __str__(self) -> str:
        return str(self.value)
