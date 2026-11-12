from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="GrowRewardCount")



@_attrs_define
class GrowRewardCount:
    """ 
        Attributes:
            count (int):
            milestone (int):
            name (str):
     """

    count: int
    milestone: int
    name: str





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        milestone = self.milestone

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "milestone": milestone,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        milestone = d.pop("milestone")

        name = d.pop("name")

        grow_reward_count = cls(
            count=count,
            milestone=milestone,
            name=name,
        )

        return grow_reward_count

