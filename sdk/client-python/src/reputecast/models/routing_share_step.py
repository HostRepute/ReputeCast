from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="RoutingShareStep")



@_attrs_define
class RoutingShareStep:
    """ 
        Attributes:
            day (int):
            share (int):
     """

    day: int
    share: int





    def to_dict(self) -> dict[str, Any]:
        day = self.day

        share = self.share


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "day": day,
            "share": share,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        day = d.pop("day")

        share = d.pop("share")

        routing_share_step = cls(
            day=day,
            share=share,
        )

        return routing_share_step

