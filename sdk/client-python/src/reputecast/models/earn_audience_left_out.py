from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="EarnAudienceLeftOut")



@_attrs_define
class EarnAudienceLeftOut:
    """ 
        Attributes:
            inactive (int):
            not_on_list (int):
            suppressed (int):
            unsubscribed (int):
     """

    inactive: int
    not_on_list: int
    suppressed: int
    unsubscribed: int





    def to_dict(self) -> dict[str, Any]:
        inactive = self.inactive

        not_on_list = self.not_on_list

        suppressed = self.suppressed

        unsubscribed = self.unsubscribed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "inactive": inactive,
            "notOnList": not_on_list,
            "suppressed": suppressed,
            "unsubscribed": unsubscribed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        inactive = d.pop("inactive")

        not_on_list = d.pop("notOnList")

        suppressed = d.pop("suppressed")

        unsubscribed = d.pop("unsubscribed")

        earn_audience_left_out = cls(
            inactive=inactive,
            not_on_list=not_on_list,
            suppressed=suppressed,
            unsubscribed=unsubscribed,
        )

        return earn_audience_left_out

