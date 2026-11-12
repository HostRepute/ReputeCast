from enum import StrEnum

class PushPromptConfigCorner(StrEnum):
    BOTTOM_LEFT = "bottom_left"
    BOTTOM_RIGHT = "bottom_right"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
