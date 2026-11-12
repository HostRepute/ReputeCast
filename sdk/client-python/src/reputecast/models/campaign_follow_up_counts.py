from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="CampaignFollowUpCounts")



@_attrs_define
class CampaignFollowUpCounts:
    """ 
        Attributes:
            clickers (int):
            machine_only (int):
            non_openers (int): No open at all
            received (int):
            subject (str):
            sent_at (datetime.datetime | Unset):
     """

    clickers: int
    machine_only: int
    non_openers: int
    received: int
    subject: str
    sent_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        clickers = self.clickers

        machine_only = self.machine_only

        non_openers = self.non_openers

        received = self.received

        subject = self.subject

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clickers": clickers,
            "machineOnly": machine_only,
            "nonOpeners": non_openers,
            "received": received,
            "subject": subject,
        })
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clickers = d.pop("clickers")

        machine_only = d.pop("machineOnly")

        non_openers = d.pop("nonOpeners")

        received = d.pop("received")

        subject = d.pop("subject")

        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        campaign_follow_up_counts = cls(
            clickers=clickers,
            machine_only=machine_only,
            non_openers=non_openers,
            received=received,
            subject=subject,
            sent_at=sent_at,
        )

        return campaign_follow_up_counts

