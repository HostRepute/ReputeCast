from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReferralTrackPoint")



@_attrs_define
class ReferralTrackPoint:
    """ 
        Attributes:
            count (int):
            reward (str):
            reached_at (datetime.datetime | Unset):
     """

    count: int
    reward: str
    reached_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        reward = self.reward

        reached_at: str | Unset = UNSET
        if not isinstance(self.reached_at, Unset):
            reached_at = self.reached_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "reward": reward,
        })
        if reached_at is not UNSET:
            field_dict["reachedAt"] = reached_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        reward = d.pop("reward")

        _reached_at = d.pop("reachedAt", UNSET)
        reached_at: datetime.datetime | Unset
        if isinstance(_reached_at,  Unset):
            reached_at = UNSET
        else:
            reached_at = datetime.datetime.fromisoformat(_reached_at)




        referral_track_point = cls(
            count=count,
            reward=reward,
            reached_at=reached_at,
        )

        return referral_track_point

