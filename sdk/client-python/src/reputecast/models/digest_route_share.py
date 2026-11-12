from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="DigestRouteShare")



@_attrs_define
class DigestRouteShare:
    """ 
        Attributes:
            adapter (str):
            name (str):
            percent (float):
            provider_id (str):
            short (str):
            status (str):
            warmup_day (int | Unset):
            warmup_days (int | Unset):
            warmup_ends (datetime.datetime | Unset):
     """

    adapter: str
    name: str
    percent: float
    provider_id: str
    short: str
    status: str
    warmup_day: int | Unset = UNSET
    warmup_days: int | Unset = UNSET
    warmup_ends: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        name = self.name

        percent = self.percent

        provider_id = self.provider_id

        short = self.short

        status = self.status

        warmup_day = self.warmup_day

        warmup_days = self.warmup_days

        warmup_ends: str | Unset = UNSET
        if not isinstance(self.warmup_ends, Unset):
            warmup_ends = self.warmup_ends.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "name": name,
            "percent": percent,
            "providerId": provider_id,
            "short": short,
            "status": status,
        })
        if warmup_day is not UNSET:
            field_dict["warmupDay"] = warmup_day
        if warmup_days is not UNSET:
            field_dict["warmupDays"] = warmup_days
        if warmup_ends is not UNSET:
            field_dict["warmupEnds"] = warmup_ends

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        name = d.pop("name")

        percent = d.pop("percent")

        provider_id = d.pop("providerId")

        short = d.pop("short")

        status = d.pop("status")

        warmup_day = d.pop("warmupDay", UNSET)

        warmup_days = d.pop("warmupDays", UNSET)

        _warmup_ends = d.pop("warmupEnds", UNSET)
        warmup_ends: datetime.datetime | Unset
        if isinstance(_warmup_ends,  Unset):
            warmup_ends = UNSET
        else:
            warmup_ends = datetime.datetime.fromisoformat(_warmup_ends)




        digest_route_share = cls(
            adapter=adapter,
            name=name,
            percent=percent,
            provider_id=provider_id,
            short=short,
            status=status,
            warmup_day=warmup_day,
            warmup_days=warmup_days,
            warmup_ends=warmup_ends,
        )

        return digest_route_share

