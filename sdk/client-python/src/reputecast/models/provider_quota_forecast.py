from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ProviderQuotaForecast")



@_attrs_define
class ProviderQuotaForecast:
    """ 
        Attributes:
            per_day (int):
            provider_id (str):
            stream (str):
            next_tier_adds (int | Unset):
            next_tier_price (float | Unset):
            runs_out_on (datetime.datetime | Unset):
     """

    per_day: int
    provider_id: str
    stream: str
    next_tier_adds: int | Unset = UNSET
    next_tier_price: float | Unset = UNSET
    runs_out_on: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        per_day = self.per_day

        provider_id = self.provider_id

        stream = self.stream

        next_tier_adds = self.next_tier_adds

        next_tier_price = self.next_tier_price

        runs_out_on: str | Unset = UNSET
        if not isinstance(self.runs_out_on, Unset):
            runs_out_on = self.runs_out_on.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "perDay": per_day,
            "providerId": provider_id,
            "stream": stream,
        })
        if next_tier_adds is not UNSET:
            field_dict["nextTierAdds"] = next_tier_adds
        if next_tier_price is not UNSET:
            field_dict["nextTierPrice"] = next_tier_price
        if runs_out_on is not UNSET:
            field_dict["runsOutOn"] = runs_out_on

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        per_day = d.pop("perDay")

        provider_id = d.pop("providerId")

        stream = d.pop("stream")

        next_tier_adds = d.pop("nextTierAdds", UNSET)

        next_tier_price = d.pop("nextTierPrice", UNSET)

        _runs_out_on = d.pop("runsOutOn", UNSET)
        runs_out_on: datetime.datetime | Unset
        if isinstance(_runs_out_on,  Unset):
            runs_out_on = UNSET
        else:
            runs_out_on = datetime.datetime.fromisoformat(_runs_out_on)




        provider_quota_forecast = cls(
            per_day=per_day,
            provider_id=provider_id,
            stream=stream,
            next_tier_adds=next_tier_adds,
            next_tier_price=next_tier_price,
            runs_out_on=runs_out_on,
        )

        return provider_quota_forecast

