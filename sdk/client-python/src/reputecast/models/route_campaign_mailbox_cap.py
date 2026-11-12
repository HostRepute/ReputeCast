from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="RouteCampaignMailboxCap")



@_attrs_define
class RouteCampaignMailboxCap:
    """ 
        Attributes:
            cap (int):
            left (int):
            mailbox (str):
            over (int):
            recipients (int):
     """

    cap: int
    left: int
    mailbox: str
    over: int
    recipients: int





    def to_dict(self) -> dict[str, Any]:
        cap = self.cap

        left = self.left

        mailbox = self.mailbox

        over = self.over

        recipients = self.recipients


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cap": cap,
            "left": left,
            "mailbox": mailbox,
            "over": over,
            "recipients": recipients,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cap = d.pop("cap")

        left = d.pop("left")

        mailbox = d.pop("mailbox")

        over = d.pop("over")

        recipients = d.pop("recipients")

        route_campaign_mailbox_cap = cls(
            cap=cap,
            left=left,
            mailbox=mailbox,
            over=over,
            recipients=recipients,
        )

        return route_campaign_mailbox_cap

