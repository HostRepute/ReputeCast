from enum import StrEnum

class AutomationNodeType(StrEnum):
    BRANCH = "branch"
    DELAY = "delay"
    EMAIL = "email"
    EXIT = "exit"
    FILTER = "filter"
    GOAL = "goal"
    LIST = "list"
    PUSH = "push"
    SMS = "sms"
    SPLIT = "split"
    TRIGGER = "trigger"
    WEBHOOK = "webhook"
    WHATSAPP = "whatsapp"

    def __str__(self) -> str:
        return str(self.value)
