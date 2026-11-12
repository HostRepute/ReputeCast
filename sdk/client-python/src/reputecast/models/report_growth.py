from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReportGrowth")



@_attrs_define
class ReportGrowth:
    """ 
        Attributes:
            forwards (int):
            new_subscribers (int):
            referrals (int):
            web_post (int):
            web_views (int):
            since (datetime.datetime | Unset):
     """

    forwards: int
    new_subscribers: int
    referrals: int
    web_post: int
    web_views: int
    since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        forwards = self.forwards

        new_subscribers = self.new_subscribers

        referrals = self.referrals

        web_post = self.web_post

        web_views = self.web_views

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "forwards": forwards,
            "newSubscribers": new_subscribers,
            "referrals": referrals,
            "webPost": web_post,
            "webViews": web_views,
        })
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        forwards = d.pop("forwards")

        new_subscribers = d.pop("newSubscribers")

        referrals = d.pop("referrals")

        web_post = d.pop("webPost")

        web_views = d.pop("webViews")

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        report_growth = cls(
            forwards=forwards,
            new_subscribers=new_subscribers,
            referrals=referrals,
            web_post=web_post,
            web_views=web_views,
            since=since,
        )

        return report_growth

