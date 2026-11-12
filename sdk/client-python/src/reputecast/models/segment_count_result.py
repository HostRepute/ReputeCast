from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.segment_tier_mix import SegmentTierMix





T = TypeVar("T", bound="SegmentCountResult")



@_attrs_define
class SegmentCountResult:
    """ 
        Attributes:
            count (int):
            generated_sql (str):
            ms (int): How long the count took
            tiers (SegmentTierMix):
            total (int): Contacts in the workspace
            cached (bool | Unset):
     """

    count: int
    generated_sql: str
    ms: int
    tiers: SegmentTierMix
    total: int
    cached: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.segment_tier_mix import SegmentTierMix # noqa: PLC0415
        count = self.count

        generated_sql = self.generated_sql

        ms = self.ms

        tiers = self.tiers.to_dict()

        total = self.total

        cached = self.cached


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "generatedSql": generated_sql,
            "ms": ms,
            "tiers": tiers,
            "total": total,
        })
        if cached is not UNSET:
            field_dict["cached"] = cached

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.segment_tier_mix import SegmentTierMix # noqa: PLC0415
        d = dict(src_dict)
        count = d.pop("count")

        generated_sql = d.pop("generatedSql")

        ms = d.pop("ms")

        tiers = SegmentTierMix.from_dict(d.pop("tiers"))




        total = d.pop("total")

        cached = d.pop("cached", UNSET)

        segment_count_result = cls(
            count=count,
            generated_sql=generated_sql,
            ms=ms,
            tiers=tiers,
            total=total,
            cached=cached,
        )

        return segment_count_result

