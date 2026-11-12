from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReferralEmailSettings")



@_attrs_define
class ReferralEmailSettings:
    """ 
        Attributes:
            milestones (int):
            on (int):
            reply_to (str):
            sender_id (str):
     """

    milestones: int
    on: int
    reply_to: str
    sender_id: str





    def to_dict(self) -> dict[str, Any]:
        milestones = self.milestones

        on = self.on

        reply_to = self.reply_to

        sender_id = self.sender_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "milestones": milestones,
            "on": on,
            "replyTo": reply_to,
            "senderId": sender_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        milestones = d.pop("milestones")

        on = d.pop("on")

        reply_to = d.pop("replyTo")

        sender_id = d.pop("senderId")

        referral_email_settings = cls(
            milestones=milestones,
            on=on,
            reply_to=reply_to,
            sender_id=sender_id,
        )

        return referral_email_settings

