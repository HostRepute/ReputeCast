from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.revenue_conversion_line_matched import RevenueConversionLineMatched
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="RevenueConversionLine")



@_attrs_define
class RevenueConversionLine:
    """ 
        Attributes:
            at (datetime.datetime):
            credited (str):
            currency (str):
            matched (RevenueConversionLineMatched):
            value_cents (int):
            campaign_id (str | Unset):
     """

    at: datetime.datetime
    credited: str
    currency: str
    matched: RevenueConversionLineMatched
    value_cents: int
    campaign_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        credited = self.credited

        currency = self.currency

        matched = self.matched.value

        value_cents = self.value_cents

        campaign_id = self.campaign_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "credited": credited,
            "currency": currency,
            "matched": matched,
            "valueCents": value_cents,
        })
        if campaign_id is not UNSET:
            field_dict["campaignId"] = campaign_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        credited = d.pop("credited")

        currency = d.pop("currency")

        matched = RevenueConversionLineMatched(d.pop("matched"))




        value_cents = d.pop("valueCents")

        campaign_id = d.pop("campaignId", UNSET)

        revenue_conversion_line = cls(
            at=at,
            credited=credited,
            currency=currency,
            matched=matched,
            value_cents=value_cents,
            campaign_id=campaign_id,
        )

        return revenue_conversion_line

