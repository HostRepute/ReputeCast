from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReferralRewardRow")



@_attrs_define
class ReferralRewardRow:
    """ 
        Attributes:
            count (int):
            failed (int):
            milestone_id (str):
            name (str):
            reached (int):
            sent (int):
            type_ (str):
            used (int):
            waiting (int):
     """

    count: int
    failed: int
    milestone_id: str
    name: str
    reached: int
    sent: int
    type_: str
    used: int
    waiting: int





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        failed = self.failed

        milestone_id = self.milestone_id

        name = self.name

        reached = self.reached

        sent = self.sent

        type_ = self.type_

        used = self.used

        waiting = self.waiting


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "failed": failed,
            "milestoneId": milestone_id,
            "name": name,
            "reached": reached,
            "sent": sent,
            "type": type_,
            "used": used,
            "waiting": waiting,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        failed = d.pop("failed")

        milestone_id = d.pop("milestoneId")

        name = d.pop("name")

        reached = d.pop("reached")

        sent = d.pop("sent")

        type_ = d.pop("type")

        used = d.pop("used")

        waiting = d.pop("waiting")

        referral_reward_row = cls(
            count=count,
            failed=failed,
            milestone_id=milestone_id,
            name=name,
            reached=reached,
            sent=sent,
            type_=type_,
            used=used,
            waiting=waiting,
        )

        return referral_reward_row

