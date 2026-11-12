from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AutomationFieldOption")



@_attrs_define
class AutomationFieldOption:
    """ 
        Attributes:
            key (str):
            label (str):
            type_ (str):
     """

    key: str
    label: str
    type_: str





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        label = self.label

        type_ = self.type_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "label": label,
            "type": type_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        label = d.pop("label")

        type_ = d.pop("type")

        automation_field_option = cls(
            key=key,
            label=label,
            type_=type_,
        )

        return automation_field_option

