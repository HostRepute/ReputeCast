from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.placement_auth_summary import PlacementAuthSummary
  from ..models.placement_counts import PlacementCounts
  from ..models.placement_timing import PlacementTiming





T = TypeVar("T", bound="PlacementProviderResult")



@_attrs_define
class PlacementProviderResult:
    """ 
        Attributes:
            adapter (str):
            auth (PlacementAuthSummary):
            color (str):
            cost (float):
            counts (PlacementCounts):
            detail (str):
            id (str):
            name (str):
            short (str):
            status (str):
            fastest (PlacementTiming | Unset):
            ip (str | Unset):
            median_ms (int | Unset):
            slowest (PlacementTiming | Unset):
            warm_day (int | Unset):
            warm_days (int | Unset):
     """

    adapter: str
    auth: PlacementAuthSummary
    color: str
    cost: float
    counts: PlacementCounts
    detail: str
    id: str
    name: str
    short: str
    status: str
    fastest: PlacementTiming | Unset = UNSET
    ip: str | Unset = UNSET
    median_ms: int | Unset = UNSET
    slowest: PlacementTiming | Unset = UNSET
    warm_day: int | Unset = UNSET
    warm_days: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.placement_auth_summary import PlacementAuthSummary # noqa: PLC0415
        from ..models.placement_counts import PlacementCounts # noqa: PLC0415
        from ..models.placement_timing import PlacementTiming # noqa: PLC0415
        adapter = self.adapter

        auth = self.auth.to_dict()

        color = self.color

        cost = self.cost

        counts = self.counts.to_dict()

        detail = self.detail

        id = self.id

        name = self.name

        short = self.short

        status = self.status

        fastest: dict[str, Any] | Unset = UNSET
        if not isinstance(self.fastest, Unset):
            fastest = self.fastest.to_dict()

        ip = self.ip

        median_ms = self.median_ms

        slowest: dict[str, Any] | Unset = UNSET
        if not isinstance(self.slowest, Unset):
            slowest = self.slowest.to_dict()

        warm_day = self.warm_day

        warm_days = self.warm_days


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "auth": auth,
            "color": color,
            "cost": cost,
            "counts": counts,
            "detail": detail,
            "id": id,
            "name": name,
            "short": short,
            "status": status,
        })
        if fastest is not UNSET:
            field_dict["fastest"] = fastest
        if ip is not UNSET:
            field_dict["ip"] = ip
        if median_ms is not UNSET:
            field_dict["medianMs"] = median_ms
        if slowest is not UNSET:
            field_dict["slowest"] = slowest
        if warm_day is not UNSET:
            field_dict["warmDay"] = warm_day
        if warm_days is not UNSET:
            field_dict["warmDays"] = warm_days

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.placement_auth_summary import PlacementAuthSummary # noqa: PLC0415
        from ..models.placement_counts import PlacementCounts # noqa: PLC0415
        from ..models.placement_timing import PlacementTiming # noqa: PLC0415
        d = dict(src_dict)
        adapter = d.pop("adapter")

        auth = PlacementAuthSummary.from_dict(d.pop("auth"))




        color = d.pop("color")

        cost = d.pop("cost")

        counts = PlacementCounts.from_dict(d.pop("counts"))




        detail = d.pop("detail")

        id = d.pop("id")

        name = d.pop("name")

        short = d.pop("short")

        status = d.pop("status")

        _fastest = d.pop("fastest", UNSET)
        fastest: PlacementTiming | Unset
        if isinstance(_fastest,  Unset):
            fastest = UNSET
        else:
            fastest = PlacementTiming.from_dict(_fastest)




        ip = d.pop("ip", UNSET)

        median_ms = d.pop("medianMs", UNSET)

        _slowest = d.pop("slowest", UNSET)
        slowest: PlacementTiming | Unset
        if isinstance(_slowest,  Unset):
            slowest = UNSET
        else:
            slowest = PlacementTiming.from_dict(_slowest)




        warm_day = d.pop("warmDay", UNSET)

        warm_days = d.pop("warmDays", UNSET)

        placement_provider_result = cls(
            adapter=adapter,
            auth=auth,
            color=color,
            cost=cost,
            counts=counts,
            detail=detail,
            id=id,
            name=name,
            short=short,
            status=status,
            fastest=fastest,
            ip=ip,
            median_ms=median_ms,
            slowest=slowest,
            warm_day=warm_day,
            warm_days=warm_days,
        )

        return placement_provider_result

