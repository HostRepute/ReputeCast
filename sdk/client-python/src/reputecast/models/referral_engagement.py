from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ReferralEngagement")



@_attrs_define
class ReferralEngagement:
    """ 
        Attributes:
            clicks (int):
            joined_after (str):
            opened (list[str]):
            short (list[str]):
     """

    clicks: int
    joined_after: str
    opened: list[str]
    short: list[str]





    def to_dict(self) -> dict[str, Any]:
        clicks = self.clicks

        joined_after = self.joined_after

        opened = self.opened



        short = self.short




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clicks": clicks,
            "joinedAfter": joined_after,
            "opened": opened,
            "short": short,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clicks = d.pop("clicks")

        joined_after = d.pop("joinedAfter")

        opened = cast(list[str], d.pop("opened"))


        short = cast(list[str], d.pop("short"))


        referral_engagement = cls(
            clicks=clicks,
            joined_after=joined_after,
            opened=opened,
            short=short,
        )

        return referral_engagement

