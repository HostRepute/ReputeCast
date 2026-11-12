from enum import StrEnum

class PushPromptConfigIos(StrEnum):
    HIDDEN = "hidden"
    STEPS = "steps"

    def __str__(self) -> str:
        return str(self.value)
