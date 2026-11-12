from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.offer_applies_input_period import OfferAppliesInputPeriod
from uuid import UUID






T = TypeVar("T", bound="OfferAppliesInput")



@_attrs_define
class OfferAppliesInput:
    """ 
        Attributes:
            period (OfferAppliesInputPeriod):
            tier_id (UUID):
     """

    period: OfferAppliesInputPeriod
    tier_id: UUID





    def to_dict(self) -> dict[str, Any]:
        period = self.period.value

        tier_id = str(self.tier_id)


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "period": period,
            "tierId": tier_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        period = OfferAppliesInputPeriod(d.pop("period"))




        tier_id = UUID(d.pop("tierId"))




        offer_applies_input = cls(
            period=period,
            tier_id=tier_id,
        )

        return offer_applies_input

