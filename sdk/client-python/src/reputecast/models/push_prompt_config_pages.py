from enum import StrEnum

class PushPromptConfigPages(StrEnum):
    ALL_EXCEPT = "all_except"
    ONLY = "only"

    def __str__(self) -> str:
        return str(self.value)
