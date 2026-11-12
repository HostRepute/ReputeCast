from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AgencyRate")



@_attrs_define
class AgencyRate:
    """ 
        Attributes:
            cost_plus_pct (float | Unset):
            flat (float | Unset):
            monthly_fee (float | Unset):
            per_1000_emails (float | Unset):
            per_segment (float | Unset):
     """

    cost_plus_pct: float | Unset = UNSET
    flat: float | Unset = UNSET
    monthly_fee: float | Unset = UNSET
    per_1000_emails: float | Unset = UNSET
    per_segment: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        cost_plus_pct = self.cost_plus_pct

        flat = self.flat

        monthly_fee = self.monthly_fee

        per_1000_emails = self.per_1000_emails

        per_segment = self.per_segment


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if cost_plus_pct is not UNSET:
            field_dict["costPlusPct"] = cost_plus_pct
        if flat is not UNSET:
            field_dict["flat"] = flat
        if monthly_fee is not UNSET:
            field_dict["monthlyFee"] = monthly_fee
        if per_1000_emails is not UNSET:
            field_dict["per1000Emails"] = per_1000_emails
        if per_segment is not UNSET:
            field_dict["perSegment"] = per_segment

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cost_plus_pct = d.pop("costPlusPct", UNSET)

        flat = d.pop("flat", UNSET)

        monthly_fee = d.pop("monthlyFee", UNSET)

        per_1000_emails = d.pop("per1000Emails", UNSET)

        per_segment = d.pop("perSegment", UNSET)

        agency_rate = cls(
            cost_plus_pct=cost_plus_pct,
            flat=flat,
            monthly_fee=monthly_fee,
            per_1000_emails=per_1000_emails,
            per_segment=per_segment,
        )

        return agency_rate

