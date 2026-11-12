from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="PushEnableInput")



@_attrs_define
class PushEnableInput:
    """ 
        Attributes:
            contact (str | Unset):
     """

    contact: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        contact = self.contact


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if contact is not UNSET:
            field_dict["contact"] = contact

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        contact = d.pop("contact", UNSET)

        push_enable_input = cls(
            contact=contact,
        )

        return push_enable_input

