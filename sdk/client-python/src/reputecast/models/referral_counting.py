from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_counting_hold_days import ReferralCountingHoldDays
from typing import cast

if TYPE_CHECKING:
  from ..models.referral_rule import ReferralRule





T = TypeVar("T", bound="ReferralCounting")



@_attrs_define
class ReferralCounting:
    """ 
        Attributes:
            burst (ReferralRule):
            disposable (bool):
            hold_days (ReferralCountingHoldDays):
            left (bool):
            numbered (bool):
            same_ip (bool):
            shared_ip (ReferralRule):
     """

    burst: ReferralRule
    disposable: bool
    hold_days: ReferralCountingHoldDays
    left: bool
    numbered: bool
    same_ip: bool
    shared_ip: ReferralRule





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_rule import ReferralRule # noqa: PLC0415
        burst = self.burst.to_dict()

        disposable = self.disposable

        hold_days = self.hold_days.value

        left = self.left

        numbered = self.numbered

        same_ip = self.same_ip

        shared_ip = self.shared_ip.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "burst": burst,
            "disposable": disposable,
            "holdDays": hold_days,
            "left": left,
            "numbered": numbered,
            "sameIp": same_ip,
            "sharedIp": shared_ip,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_rule import ReferralRule # noqa: PLC0415
        d = dict(src_dict)
        burst = ReferralRule.from_dict(d.pop("burst"))




        disposable = d.pop("disposable")

        hold_days = ReferralCountingHoldDays(d.pop("holdDays"))




        left = d.pop("left")

        numbered = d.pop("numbered")

        same_ip = d.pop("sameIp")

        shared_ip = ReferralRule.from_dict(d.pop("sharedIp"))




        referral_counting = cls(
            burst=burst,
            disposable=disposable,
            hold_days=hold_days,
            left=left,
            numbered=numbered,
            same_ip=same_ip,
            shared_ip=shared_ip,
        )

        return referral_counting

