from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.sms_failure_code import SmsFailureCode





T = TypeVar("T", bound="SmsFailureReason")



@_attrs_define
class SmsFailureReason:
    """ 
        Attributes:
            codes (list[SmsFailureCode]):
            count (int):
            limit (float):
            rate (float):
            reason (str):
     """

    codes: list[SmsFailureCode]
    count: int
    limit: float
    rate: float
    reason: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_failure_code import SmsFailureCode # noqa: PLC0415
        codes = []
        for codes_item_data in self.codes:
            codes_item = codes_item_data.to_dict()
            codes.append(codes_item)



        count = self.count

        limit = self.limit

        rate = self.rate

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "codes": codes,
            "count": count,
            "limit": limit,
            "rate": rate,
            "reason": reason,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_failure_code import SmsFailureCode # noqa: PLC0415
        d = dict(src_dict)
        codes = []
        _codes = d.pop("codes")
        for codes_item_data in (_codes):
            codes_item = SmsFailureCode.from_dict(codes_item_data)



            codes.append(codes_item)


        count = d.pop("count")

        limit = d.pop("limit")

        rate = d.pop("rate")

        reason = d.pop("reason")

        sms_failure_reason = cls(
            codes=codes,
            count=count,
            limit=limit,
            rate=rate,
            reason=reason,
        )

        return sms_failure_reason

