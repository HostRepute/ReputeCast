from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="WebsiteNumbers")



@_attrs_define
class WebsiteNumbers:
    """ 
        Attributes:
            from_ (datetime.datetime):
            signups (int):
            until (datetime.datetime):
            upgrades (int):
            views (int):
            visitors (int):
     """

    from_: datetime.datetime
    signups: int
    until: datetime.datetime
    upgrades: int
    views: int
    visitors: int





    def to_dict(self) -> dict[str, Any]:
        from_ = self.from_.isoformat()

        signups = self.signups

        until = self.until.isoformat()

        upgrades = self.upgrades

        views = self.views

        visitors = self.visitors


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "from": from_,
            "signups": signups,
            "until": until,
            "upgrades": upgrades,
            "views": views,
            "visitors": visitors,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        from_ = datetime.datetime.fromisoformat(d.pop("from"))




        signups = d.pop("signups")

        until = datetime.datetime.fromisoformat(d.pop("until"))




        upgrades = d.pop("upgrades")

        views = d.pop("views")

        visitors = d.pop("visitors")

        website_numbers = cls(
            from_=from_,
            signups=signups,
            until=until,
            upgrades=upgrades,
            views=views,
            visitors=visitors,
        )

        return website_numbers

