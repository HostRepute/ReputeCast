from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_limit_rate_reason import ProviderLimitRateReason
from ..types import UNSET, Unset






T = TypeVar("T", bound="ProviderLimitRate")



@_attrs_define
class ProviderLimitRate:
    """ 
        Attributes:
            limit (float):
            stale (bool):
            label (str | Unset):
            pause (float | Unset):
            reason (ProviderLimitRateReason | Unset):
            review (float | Unset):
            value (float | Unset): Percent over the last 7 days
     """

    limit: float
    stale: bool
    label: str | Unset = UNSET
    pause: float | Unset = UNSET
    reason: ProviderLimitRateReason | Unset = UNSET
    review: float | Unset = UNSET
    value: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        limit = self.limit

        stale = self.stale

        label = self.label

        pause = self.pause

        reason: str | Unset = UNSET
        if not isinstance(self.reason, Unset):
            reason = self.reason.value


        review = self.review

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "limit": limit,
            "stale": stale,
        })
        if label is not UNSET:
            field_dict["label"] = label
        if pause is not UNSET:
            field_dict["pause"] = pause
        if reason is not UNSET:
            field_dict["reason"] = reason
        if review is not UNSET:
            field_dict["review"] = review
        if value is not UNSET:
            field_dict["value"] = value

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        limit = d.pop("limit")

        stale = d.pop("stale")

        label = d.pop("label", UNSET)

        pause = d.pop("pause", UNSET)

        _reason = d.pop("reason", UNSET)
        reason: ProviderLimitRateReason | Unset
        if isinstance(_reason,  Unset):
            reason = UNSET
        else:
            reason = ProviderLimitRateReason(_reason)




        review = d.pop("review", UNSET)

        value = d.pop("value", UNSET)

        provider_limit_rate = cls(
            limit=limit,
            stale=stale,
            label=label,
            pause=pause,
            reason=reason,
            review=review,
            value=value,
        )

        return provider_limit_rate

