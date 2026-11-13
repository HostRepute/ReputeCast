from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.signup_form_step import SignupFormStep





T = TypeVar("T", bound="SignupFormVariantContent")



@_attrs_define
class SignupFormVariantContent:
    """ 
        Attributes:
            button (str):
            description (str):
            heading (str):
            popup_delay (int):
            popup_scroll (int):
            after_submit_text (str | Unset):
            popup_exit (bool | Unset):
            steps (list[SignupFormStep] | Unset): Omitted or one step: everything at once
     """

    button: str
    description: str
    heading: str
    popup_delay: int
    popup_scroll: int
    after_submit_text: str | Unset = UNSET
    popup_exit: bool | Unset = UNSET
    steps: list[SignupFormStep] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.signup_form_step import SignupFormStep # noqa: PLC0415
        button = self.button

        description = self.description

        heading = self.heading

        popup_delay = self.popup_delay

        popup_scroll = self.popup_scroll

        after_submit_text = self.after_submit_text

        popup_exit = self.popup_exit

        steps: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.steps, Unset):
            steps = []
            for steps_item_data in self.steps:
                steps_item = steps_item_data.to_dict()
                steps.append(steps_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "button": button,
            "description": description,
            "heading": heading,
            "popupDelay": popup_delay,
            "popupScroll": popup_scroll,
        })
        if after_submit_text is not UNSET:
            field_dict["afterSubmitText"] = after_submit_text
        if popup_exit is not UNSET:
            field_dict["popupExit"] = popup_exit
        if steps is not UNSET:
            field_dict["steps"] = steps

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.signup_form_step import SignupFormStep # noqa: PLC0415
        d = dict(src_dict)
        button = d.pop("button")

        description = d.pop("description")

        heading = d.pop("heading")

        popup_delay = d.pop("popupDelay")

        popup_scroll = d.pop("popupScroll")

        after_submit_text = d.pop("afterSubmitText", UNSET)

        popup_exit = d.pop("popupExit", UNSET)

        _steps = d.pop("steps", UNSET)
        steps: list[SignupFormStep] | Unset = UNSET
        if _steps is not UNSET:
            steps = []
            for steps_item_data in _steps:
                steps_item = SignupFormStep.from_dict(steps_item_data)



                steps.append(steps_item)


        signup_form_variant_content = cls(
            button=button,
            description=description,
            heading=heading,
            popup_delay=popup_delay,
            popup_scroll=popup_scroll,
            after_submit_text=after_submit_text,
            popup_exit=popup_exit,
            steps=steps,
        )

        return signup_form_variant_content

