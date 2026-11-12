from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_reason_count_reason import ReferralReasonCountReason






T = TypeVar("T", bound="ReferralReasonCount")



@_attrs_define
class ReferralReasonCount:
    """ 
        Attributes:
            count (int):
            left_kind (str):
            reason (ReferralReasonCountReason):
     """

    count: int
    left_kind: str
    reason: ReferralReasonCountReason





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        left_kind = self.left_kind

        reason = self.reason.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "leftKind": left_kind,
            "reason": reason,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        left_kind = d.pop("leftKind")

        reason = ReferralReasonCountReason(d.pop("reason"))




        referral_reason_count = cls(
            count=count,
            left_kind=left_kind,
            reason=reason,
        )

        return referral_reason_count

