from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ProviderSendRate")



@_attrs_define
class ProviderSendRate:
    """ 
        Attributes:
            max_ (float):
            now (float):
            peak_today (float):
            throttled (bool):
            label (str | Unset):
     """

    max_: float
    now: float
    peak_today: float
    throttled: bool
    label: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        max_ = self.max_

        now = self.now

        peak_today = self.peak_today

        throttled = self.throttled

        label = self.label


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "max": max_,
            "now": now,
            "peakToday": peak_today,
            "throttled": throttled,
        })
        if label is not UNSET:
            field_dict["label"] = label

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        max_ = d.pop("max")

        now = d.pop("now")

        peak_today = d.pop("peakToday")

        throttled = d.pop("throttled")

        label = d.pop("label", UNSET)

        provider_send_rate = cls(
            max_=max_,
            now=now,
            peak_today=peak_today,
            throttled=throttled,
            label=label,
        )

        return provider_send_rate

