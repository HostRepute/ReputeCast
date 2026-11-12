from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.custom_field_input_display import CustomFieldInputDisplay
from ..models.custom_field_input_type import CustomFieldInputType
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="CustomFieldInput")



@_attrs_define
class CustomFieldInput:
    """ 
        Attributes:
            label (str):
            display (CustomFieldInputDisplay | Unset):
            key (str | Unset): Lowercase letters, digits and _; fixed after creation
            options (list[str] | Unset):
            type_ (CustomFieldInputType | Unset):
     """

    label: str
    display: CustomFieldInputDisplay | Unset = UNSET
    key: str | Unset = UNSET
    options: list[str] | Unset = UNSET
    type_: CustomFieldInputType | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        label = self.label

        display: str | Unset = UNSET
        if not isinstance(self.display, Unset):
            display = self.display.value


        key = self.key

        options: list[str] | Unset = UNSET
        if not isinstance(self.options, Unset):
            options = self.options



        type_: str | Unset = UNSET
        if not isinstance(self.type_, Unset):
            type_ = self.type_.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "label": label,
        })
        if display is not UNSET:
            field_dict["display"] = display
        if key is not UNSET:
            field_dict["key"] = key
        if options is not UNSET:
            field_dict["options"] = options
        if type_ is not UNSET:
            field_dict["type"] = type_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        label = d.pop("label")

        _display = d.pop("display", UNSET)
        display: CustomFieldInputDisplay | Unset
        if isinstance(_display,  Unset):
            display = UNSET
        else:
            display = CustomFieldInputDisplay(_display)




        key = d.pop("key", UNSET)

        options = cast(list[str], d.pop("options", UNSET))


        _type_ = d.pop("type", UNSET)
        type_: CustomFieldInputType | Unset
        if isinstance(_type_,  Unset):
            type_ = UNSET
        else:
            type_ = CustomFieldInputType(_type_)




        custom_field_input = cls(
            label=label,
            display=display,
            key=key,
            options=options,
            type_=type_,
        )

        return custom_field_input

