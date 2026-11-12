from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_push_step_outside import AutomationPushStepOutside
from ..types import UNSET, Unset






T = TypeVar("T", bound="AutomationPushStep")



@_attrs_define
class AutomationPushStep:
    """ 
        Attributes:
            button (str | Unset):
            expiry_minutes (int | Unset):
            link (str | Unset):
            message (str | Unset):
            outside (AutomationPushStepOutside | Unset):
            title (str | Unset):
     """

    button: str | Unset = UNSET
    expiry_minutes: int | Unset = UNSET
    link: str | Unset = UNSET
    message: str | Unset = UNSET
    outside: AutomationPushStepOutside | Unset = UNSET
    title: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        button = self.button

        expiry_minutes = self.expiry_minutes

        link = self.link

        message = self.message

        outside: str | Unset = UNSET
        if not isinstance(self.outside, Unset):
            outside = self.outside.value


        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if button is not UNSET:
            field_dict["button"] = button
        if expiry_minutes is not UNSET:
            field_dict["expiryMinutes"] = expiry_minutes
        if link is not UNSET:
            field_dict["link"] = link
        if message is not UNSET:
            field_dict["message"] = message
        if outside is not UNSET:
            field_dict["outside"] = outside
        if title is not UNSET:
            field_dict["title"] = title

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        button = d.pop("button", UNSET)

        expiry_minutes = d.pop("expiryMinutes", UNSET)

        link = d.pop("link", UNSET)

        message = d.pop("message", UNSET)

        _outside = d.pop("outside", UNSET)
        outside: AutomationPushStepOutside | Unset
        if isinstance(_outside,  Unset):
            outside = UNSET
        else:
            outside = AutomationPushStepOutside(_outside)




        title = d.pop("title", UNSET)

        automation_push_step = cls(
            button=button,
            expiry_minutes=expiry_minutes,
            link=link,
            message=message,
            outside=outside,
            title=title,
        )

        return automation_push_step

