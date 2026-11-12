from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WebsitePostCounts")



@_attrs_define
class WebsitePostCounts:
    """ 
        Attributes:
            all_ (int):
            draft (int):
            from_campaigns (int):
            members_only (int):
            paid (int):
            published (int):
            scheduled (int):
            unpublished (int):
            web_only (int):
     """

    all_: int
    draft: int
    from_campaigns: int
    members_only: int
    paid: int
    published: int
    scheduled: int
    unpublished: int
    web_only: int





    def to_dict(self) -> dict[str, Any]:
        all_ = self.all_

        draft = self.draft

        from_campaigns = self.from_campaigns

        members_only = self.members_only

        paid = self.paid

        published = self.published

        scheduled = self.scheduled

        unpublished = self.unpublished

        web_only = self.web_only


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "all": all_,
            "draft": draft,
            "fromCampaigns": from_campaigns,
            "membersOnly": members_only,
            "paid": paid,
            "published": published,
            "scheduled": scheduled,
            "unpublished": unpublished,
            "webOnly": web_only,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        all_ = d.pop("all")

        draft = d.pop("draft")

        from_campaigns = d.pop("fromCampaigns")

        members_only = d.pop("membersOnly")

        paid = d.pop("paid")

        published = d.pop("published")

        scheduled = d.pop("scheduled")

        unpublished = d.pop("unpublished")

        web_only = d.pop("webOnly")

        website_post_counts = cls(
            all_=all_,
            draft=draft,
            from_campaigns=from_campaigns,
            members_only=members_only,
            paid=paid,
            published=published,
            scheduled=scheduled,
            unpublished=unpublished,
            web_only=web_only,
        )

        return website_post_counts

