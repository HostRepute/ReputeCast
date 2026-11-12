from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_timeline_event_kind import ReferralTimelineEventKind
from typing import cast
import datetime






T = TypeVar("T", bound="ReferralTimelineEvent")



@_attrs_define
class ReferralTimelineEvent:
    """ 
        Attributes:
            at (datetime.datetime):
            counted (int):
            kind (ReferralTimelineEventKind):
            milestone (int):
            partner (str):
            referrer (str):
            referrer_id (str):
            reward (str):
     """

    at: datetime.datetime
    counted: int
    kind: ReferralTimelineEventKind
    milestone: int
    partner: str
    referrer: str
    referrer_id: str
    reward: str





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        counted = self.counted

        kind = self.kind.value

        milestone = self.milestone

        partner = self.partner

        referrer = self.referrer

        referrer_id = self.referrer_id

        reward = self.reward


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "counted": counted,
            "kind": kind,
            "milestone": milestone,
            "partner": partner,
            "referrer": referrer,
            "referrerId": referrer_id,
            "reward": reward,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        counted = d.pop("counted")

        kind = ReferralTimelineEventKind(d.pop("kind"))




        milestone = d.pop("milestone")

        partner = d.pop("partner")

        referrer = d.pop("referrer")

        referrer_id = d.pop("referrerId")

        reward = d.pop("reward")

        referral_timeline_event = cls(
            at=at,
            counted=counted,
            kind=kind,
            milestone=milestone,
            partner=partner,
            referrer=referrer,
            referrer_id=referrer_id,
            reward=reward,
        )

        return referral_timeline_event

