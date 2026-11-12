from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WarmupStep")



@_attrs_define
class WarmupStep:
    """ 
        Attributes:
            day (int):
            value (float):
     """

    day: int
    value: float





    def to_dict(self) -> dict[str, Any]:
        day = self.day

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "day": day,
            "value": value,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        day = d.pop("day")

        value = d.pop("value")

        warmup_step = cls(
            day=day,
            value=value,
        )

        return warmup_step

