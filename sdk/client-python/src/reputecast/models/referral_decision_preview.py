from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReferralDecisionPreview")



@_attrs_define
class ReferralDecisionPreview:
    """ 
        Attributes:
            after (int):
            count (int):
            counted (int):
            decision (str):
            now (bool):
            referrer (str):
            rewards (list[str]):
            hold_ends (datetime.datetime | Unset):
     """

    after: int
    count: int
    counted: int
    decision: str
    now: bool
    referrer: str
    rewards: list[str]
    hold_ends: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        after = self.after

        count = self.count

        counted = self.counted

        decision = self.decision

        now = self.now

        referrer = self.referrer

        rewards = self.rewards



        hold_ends: str | Unset = UNSET
        if not isinstance(self.hold_ends, Unset):
            hold_ends = self.hold_ends.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "after": after,
            "count": count,
            "counted": counted,
            "decision": decision,
            "now": now,
            "referrer": referrer,
            "rewards": rewards,
        })
        if hold_ends is not UNSET:
            field_dict["holdEnds"] = hold_ends

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        after = d.pop("after")

        count = d.pop("count")

        counted = d.pop("counted")

        decision = d.pop("decision")

        now = d.pop("now")

        referrer = d.pop("referrer")

        rewards = cast(list[str], d.pop("rewards"))


        _hold_ends = d.pop("holdEnds", UNSET)
        hold_ends: datetime.datetime | Unset
        if isinstance(_hold_ends,  Unset):
            hold_ends = UNSET
        else:
            hold_ends = datetime.datetime.fromisoformat(_hold_ends)




        referral_decision_preview = cls(
            after=after,
            count=count,
            counted=counted,
            decision=decision,
            now=now,
            referrer=referrer,
            rewards=rewards,
            hold_ends=hold_ends,
        )

        return referral_decision_preview

