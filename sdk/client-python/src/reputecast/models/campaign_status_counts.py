from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="CampaignStatusCounts")



@_attrs_define
class CampaignStatusCounts:
    """ 
        Attributes:
            all_ (int):
            archived (int):
            deleted (int):
            drafts (int):
            paid_only (int):
            scheduled (int):
            sending (int):
            sent (int):
     """

    all_: int
    archived: int
    deleted: int
    drafts: int
    paid_only: int
    scheduled: int
    sending: int
    sent: int





    def to_dict(self) -> dict[str, Any]:
        all_ = self.all_

        archived = self.archived

        deleted = self.deleted

        drafts = self.drafts

        paid_only = self.paid_only

        scheduled = self.scheduled

        sending = self.sending

        sent = self.sent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "all": all_,
            "archived": archived,
            "deleted": deleted,
            "drafts": drafts,
            "paidOnly": paid_only,
            "scheduled": scheduled,
            "sending": sending,
            "sent": sent,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        all_ = d.pop("all")

        archived = d.pop("archived")

        deleted = d.pop("deleted")

        drafts = d.pop("drafts")

        paid_only = d.pop("paidOnly")

        scheduled = d.pop("scheduled")

        sending = d.pop("sending")

        sent = d.pop("sent")

        campaign_status_counts = cls(
            all_=all_,
            archived=archived,
            deleted=deleted,
            drafts=drafts,
            paid_only=paid_only,
            scheduled=scheduled,
            sending=sending,
            sent=sent,
        )

        return campaign_status_counts

