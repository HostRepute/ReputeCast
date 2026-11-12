from enum import StrEnum

class TransactionalCheckFixKind(StrEnum):
    ADD_TEST_VALUE = "add_test_value"
    DECLARE_VARIABLE = "declare_variable"
    EDIT_LINK = "edit_link"
    INSERT_FOOTER = "insert_footer"
    MAKE_OPTIONAL = "make_optional"
    MINIFY = "minify"
    MOVE_TO_BODY = "move_to_body"
    REMOVE_VARIABLE = "remove_variable"
    REPLACE_LINK = "replace_link"
    SET_ALT = "set_alt"
    TURN_OFF_TRACKING = "turn_off_tracking"

    def __str__(self) -> str:
        return str(self.value)
