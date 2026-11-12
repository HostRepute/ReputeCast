from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.referral_reason_count import ReferralReasonCount





T = TypeVar("T", bound="ReferralFunnel")



@_attrs_define
class ReferralFunnel:
    """ 
        Attributes:
            confirmed (int):
            counted (int):
            held (int):
            in_hold (int):
            not_counted (int):
            reasons (list[ReferralReasonCount]):
            referrers (int):
            rewards_sent (int):
            signups (int):
            unconfirmed (int): Never confirmed, or still waiting to confirm
            visits (int):
            weeks (int):
            hold_ends_by (datetime.datetime | Unset):
     """

    confirmed: int
    counted: int
    held: int
    in_hold: int
    not_counted: int
    reasons: list[ReferralReasonCount]
    referrers: int
    rewards_sent: int
    signups: int
    unconfirmed: int
    visits: int
    weeks: int
    hold_ends_by: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_reason_count import ReferralReasonCount # noqa: PLC0415
        confirmed = self.confirmed

        counted = self.counted

        held = self.held

        in_hold = self.in_hold

        not_counted = self.not_counted

        reasons = []
        for reasons_item_data in self.reasons:
            reasons_item = reasons_item_data.to_dict()
            reasons.append(reasons_item)



        referrers = self.referrers

        rewards_sent = self.rewards_sent

        signups = self.signups

        unconfirmed = self.unconfirmed

        visits = self.visits

        weeks = self.weeks

        hold_ends_by: str | Unset = UNSET
        if not isinstance(self.hold_ends_by, Unset):
            hold_ends_by = self.hold_ends_by.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "confirmed": confirmed,
            "counted": counted,
            "held": held,
            "inHold": in_hold,
            "notCounted": not_counted,
            "reasons": reasons,
            "referrers": referrers,
            "rewardsSent": rewards_sent,
            "signups": signups,
            "unconfirmed": unconfirmed,
            "visits": visits,
            "weeks": weeks,
        })
        if hold_ends_by is not UNSET:
            field_dict["holdEndsBy"] = hold_ends_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_reason_count import ReferralReasonCount # noqa: PLC0415
        d = dict(src_dict)
        confirmed = d.pop("confirmed")

        counted = d.pop("counted")

        held = d.pop("held")

        in_hold = d.pop("inHold")

        not_counted = d.pop("notCounted")

        reasons = []
        _reasons = d.pop("reasons")
        for reasons_item_data in (_reasons):
            reasons_item = ReferralReasonCount.from_dict(reasons_item_data)



            reasons.append(reasons_item)


        referrers = d.pop("referrers")

        rewards_sent = d.pop("rewardsSent")

        signups = d.pop("signups")

        unconfirmed = d.pop("unconfirmed")

        visits = d.pop("visits")

        weeks = d.pop("weeks")

        _hold_ends_by = d.pop("holdEndsBy", UNSET)
        hold_ends_by: datetime.datetime | Unset
        if isinstance(_hold_ends_by,  Unset):
            hold_ends_by = UNSET
        else:
            hold_ends_by = datetime.datetime.fromisoformat(_hold_ends_by)




        referral_funnel = cls(
            confirmed=confirmed,
            counted=counted,
            held=held,
            in_hold=in_hold,
            not_counted=not_counted,
            reasons=reasons,
            referrers=referrers,
            rewards_sent=rewards_sent,
            signups=signups,
            unconfirmed=unconfirmed,
            visits=visits,
            weeks=weeks,
            hold_ends_by=hold_ends_by,
        )

        return referral_funnel

