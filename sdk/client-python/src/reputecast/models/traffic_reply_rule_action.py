from enum import StrEnum

class TrafficReplyRuleAction(StrEnum):
    ALERT = "alert"
    PAUSE = "pause"
    PAUSE_ROUTE = "pause_route"
    SLOW = "slow"

    def __str__(self) -> str:
        return str(self.value)
