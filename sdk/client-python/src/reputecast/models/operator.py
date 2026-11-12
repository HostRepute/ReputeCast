from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="Operator")



@_attrs_define
class Operator:
    """ 
        Attributes:
            key (str):
            no_value (bool | Unset):
     """

    key: str
    no_value: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        no_value = self.no_value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
        })
        if no_value is not UNSET:
            field_dict["noValue"] = no_value

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        no_value = d.pop("noValue", UNSET)

        operator = cls(
            key=key,
            no_value=no_value,
        )

        return operator

