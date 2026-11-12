from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReferralMilestoneStats")



@_attrs_define
class ReferralMilestoneStats:
    """ 
        Attributes:
            failed (int):
            reached (int):
            sent (int):
            to_fulfill (int):
            used (int):
            waiting (int):
            waiting_codes (int):
     """

    failed: int
    reached: int
    sent: int
    to_fulfill: int
    used: int
    waiting: int
    waiting_codes: int





    def to_dict(self) -> dict[str, Any]:
        failed = self.failed

        reached = self.reached

        sent = self.sent

        to_fulfill = self.to_fulfill

        used = self.used

        waiting = self.waiting

        waiting_codes = self.waiting_codes


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "failed": failed,
            "reached": reached,
            "sent": sent,
            "toFulfill": to_fulfill,
            "used": used,
            "waiting": waiting,
            "waitingCodes": waiting_codes,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        failed = d.pop("failed")

        reached = d.pop("reached")

        sent = d.pop("sent")

        to_fulfill = d.pop("toFulfill")

        used = d.pop("used")

        waiting = d.pop("waiting")

        waiting_codes = d.pop("waitingCodes")

        referral_milestone_stats = cls(
            failed=failed,
            reached=reached,
            sent=sent,
            to_fulfill=to_fulfill,
            used=used,
            waiting=waiting,
            waiting_codes=waiting_codes,
        )

        return referral_milestone_stats

