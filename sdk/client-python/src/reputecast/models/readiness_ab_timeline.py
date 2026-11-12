from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.readiness_ab_timeline_kind import ReadinessABTimelineKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReadinessABTimeline")



@_attrs_define
class ReadinessABTimeline:
    """ 
        Attributes:
            hours (int):
            kind (ReadinessABTimelineKind):
            per_variant (int):
            sample (int):
            sample_end (datetime.datetime):
            test_percent (int):
            variants (int):
            decide_at (datetime.datetime | Unset):
     """

    hours: int
    kind: ReadinessABTimelineKind
    per_variant: int
    sample: int
    sample_end: datetime.datetime
    test_percent: int
    variants: int
    decide_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        hours = self.hours

        kind = self.kind.value

        per_variant = self.per_variant

        sample = self.sample

        sample_end = self.sample_end.isoformat()

        test_percent = self.test_percent

        variants = self.variants

        decide_at: str | Unset = UNSET
        if not isinstance(self.decide_at, Unset):
            decide_at = self.decide_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "hours": hours,
            "kind": kind,
            "perVariant": per_variant,
            "sample": sample,
            "sampleEnd": sample_end,
            "testPercent": test_percent,
            "variants": variants,
        })
        if decide_at is not UNSET:
            field_dict["decideAt"] = decide_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        hours = d.pop("hours")

        kind = ReadinessABTimelineKind(d.pop("kind"))




        per_variant = d.pop("perVariant")

        sample = d.pop("sample")

        sample_end = datetime.datetime.fromisoformat(d.pop("sampleEnd"))




        test_percent = d.pop("testPercent")

        variants = d.pop("variants")

        _decide_at = d.pop("decideAt", UNSET)
        decide_at: datetime.datetime | Unset
        if isinstance(_decide_at,  Unset):
            decide_at = UNSET
        else:
            decide_at = datetime.datetime.fromisoformat(_decide_at)




        readiness_ab_timeline = cls(
            hours=hours,
            kind=kind,
            per_variant=per_variant,
            sample=sample,
            sample_end=sample_end,
            test_percent=test_percent,
            variants=variants,
            decide_at=decide_at,
        )

        return readiness_ab_timeline

