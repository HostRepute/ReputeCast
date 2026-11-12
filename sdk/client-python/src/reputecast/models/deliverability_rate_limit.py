from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_rate_limit_tone import DeliverabilityRateLimitTone
from ..types import UNSET, Unset






T = TypeVar("T", bound="DeliverabilityRateLimit")



@_attrs_define
class DeliverabilityRateLimit:
    """ 
        Attributes:
            label (str): Bounce rate, Complaint rate, Spam complaint rate
            thresholds (str):
            tone (DeliverabilityRateLimitTone):
            usage (float):
            value (float | Unset):
     """

    label: str
    thresholds: str
    tone: DeliverabilityRateLimitTone
    usage: float
    value: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        label = self.label

        thresholds = self.thresholds

        tone = self.tone.value

        usage = self.usage

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "label": label,
            "thresholds": thresholds,
            "tone": tone,
            "usage": usage,
        })
        if value is not UNSET:
            field_dict["value"] = value

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        label = d.pop("label")

        thresholds = d.pop("thresholds")

        tone = DeliverabilityRateLimitTone(d.pop("tone"))




        usage = d.pop("usage")

        value = d.pop("value", UNSET)

        deliverability_rate_limit = cls(
            label=label,
            thresholds=thresholds,
            tone=tone,
            usage=usage,
            value=value,
        )

        return deliverability_rate_limit

