from enum import StrEnum

class AutomationListActionKind(StrEnum):
    ADD_LIST = "add_list"
    ADD_TAG = "add_tag"
    REMOVE_LIST = "remove_list"
    REMOVE_TAG = "remove_tag"
    UNSUBSCRIBE_ALL = "unsubscribe_all"

    def __str__(self) -> str:
        return str(self.value)
