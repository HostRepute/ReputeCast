from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_pricing_input_model import ProviderPricingInputModel
from ..types import UNSET, Unset






T = TypeVar("T", bound="ProviderPricingInput")



@_attrs_define
class ProviderPricingInput:
    """ 
        Attributes:
            model (ProviderPricingInputModel):
            monthly (float | Unset):
            next_tier_adds (int | Unset):
            next_tier_price (float | Unset):
            per1000 (float | Unset):
     """

    model: ProviderPricingInputModel
    monthly: float | Unset = UNSET
    next_tier_adds: int | Unset = UNSET
    next_tier_price: float | Unset = UNSET
    per1000: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        model = self.model.value

        monthly = self.monthly

        next_tier_adds = self.next_tier_adds

        next_tier_price = self.next_tier_price

        per1000 = self.per1000


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "model": model,
        })
        if monthly is not UNSET:
            field_dict["monthly"] = monthly
        if next_tier_adds is not UNSET:
            field_dict["nextTierAdds"] = next_tier_adds
        if next_tier_price is not UNSET:
            field_dict["nextTierPrice"] = next_tier_price
        if per1000 is not UNSET:
            field_dict["per1000"] = per1000

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        model = ProviderPricingInputModel(d.pop("model"))




        monthly = d.pop("monthly", UNSET)

        next_tier_adds = d.pop("nextTierAdds", UNSET)

        next_tier_price = d.pop("nextTierPrice", UNSET)

        per1000 = d.pop("per1000", UNSET)

        provider_pricing_input = cls(
            model=model,
            monthly=monthly,
            next_tier_adds=next_tier_adds,
            next_tier_price=next_tier_price,
            per1000=per1000,
        )

        return provider_pricing_input

