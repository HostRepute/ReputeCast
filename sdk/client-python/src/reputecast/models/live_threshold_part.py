from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="LiveThresholdPart")



@_attrs_define
class LiveThresholdPart:
    """ 
        Attributes:
            name (str):
            key (str | Unset):
            value (float | Unset):
     """

    name: str
    key: str | Unset = UNSET
    value: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        key = self.key

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
        })
        if key is not UNSET:
            field_dict["key"] = key
        if value is not UNSET:
            field_dict["value"] = value

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        key = d.pop("key", UNSET)

        value = d.pop("value", UNSET)

        live_threshold_part = cls(
            name=name,
            key=key,
            value=value,
        )

        return live_threshold_part

